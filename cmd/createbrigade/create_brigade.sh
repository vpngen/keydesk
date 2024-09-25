#!/bin/sh

### Create brigades

# * Check if brigade already exists
# * Create system user
# * Create homedir

# * Create json datafile
# * Create special brigadier wg-user

# * Activate keydesk systemD units

# * Send brigadier config

# creating brigade and brigadier app.

# [ ${FLOCKER} != $0 ] && exec env FLOCKER="$0" flock -e "$0" "$0" $@ ||
spinlock="${TMPDIR:-/tmp}/vgbrigade.spinlock"
# shellcheck disable=SC2064
trap "rm -f '${spinlock}' 2>/dev/null" EXIT
while [ -f "${spinlock}" ]; do
    sleep 0.1
done
touch "${spinlock}" 2>/dev/null

set -e

STATS_DIR="/var/lib/vgstats"
ROUTER_SOCKETS_DIR="/var/lib/dcapi"
BRIGADE_MAKER_APP_PATH="/opt/vgkeydesk/createbrigade"
KEYDESK_APP_PATH="/opt/vgkeydesk/keydesk"

VGCERT_GROUP="vgcert"
VGSTATS_GROUP="vgstats"
VGROUTER_GROUP="vgrouter"

MODE_BRIGADE="brigade"
MODE_VGSOCKET="vgsocket"
DEFAULT_MAXUSERS=255

if [ "root" != "$(whoami)" ]; then
        echo "DEBUG EXECUTION" >&2
        DEBUG="yes"
fi

fatal() {
        cat << EOF | awk -v chunked="${chunked}" 'BEGIN {ORS=""; if (chunked != "") print length($0) "\r\n" $0 "\r\n0\r\n\r\n"; else print $0}'
{
        "code": $1,
        "desc": "$2"
        "status": "error",
        "message": "$3"
}
EOF
        exit 1
}

printdef () {
        msg="$1"

        echo "Usage: $0 -id <brigabe_id_encoded> -ep4 <endpoint IPv4> -int4 <CGNAT IPv4> -int6 <IPv6 ULA> -dns4 <DNS IPv4> -dns6 <DNS IPv6> -kd6 <keydesk IPv6> -name <B1rigadier Name :: base64> -person <Person Name :: base64> -desc <Person Desc :: base64> -url <Person URL :: base64> [-ch] [-j]" >&2

        fatal "400" "Bad request" "$msg"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        -id)
                brigade_id="$2"
                shift 2
                ;;
        -ch)
                chunked="-ch"
                shift 1
                ;;
        -j)
                json="-j"
                shift 1
                ;;
        -d)
                if [ -z "$DEBUG" ]; then
                        printdef "The '-d' option is only for debug"
                fi

                DB_DIR="$2"
                shift 2
                ;;
        -c)
                if [ -z "$DEBUG" ]; then
                        printdef "The '-c' option is only for debug"
                fi

                CONF_DIR="$2"
                shift 2
                ;;
        -a)
                if [ -z "$DEBUG" ]; then
                        printdef "The '-a' option is only for debug"
                fi

                apiaddr="-a $2"
                shift 2
                ;;
        -wg)
                wg_configs="-wg $2"
                shift 2
                ;;
        -ipsec)
                ipsec_configs="-ipsec $2"
                shift 2
                ;;
        -ovc)
                ovc_configs="-ovc $2"
                shift 2
                ;;
        -outline)
                outline_configs="-outline $2"
                shift 2
                ;;
        -ep4)
                endpoint_ip4="$2"
                shift 2
                ;;
        -int4)
                ip4_cgnat="$2"
                shift 2
                ;;
        -int6)
                ip6_ula="$2"
                shift 2
                ;;
        -dns4)
                dns_ip4="$2"
                shift 2
                ;;
        -dns6)
                dns_ip6="$2"
                shift 2
                ;;
        -kd6)
                keydesk_ip6="$2"
                shift 2
                ;;
        -name)
                brigadier_name="$2"
                shift 2
                ;;
        -person)
                person_name="$2"
                shift 2
                ;;
        -desc)
                person_desc="$2"
                shift 2
                ;;
        -url)
                person_url="$2"
                shift 2
                ;;
        -p)
                port="$2"
                shift 2

                case "${port}" in
                        [0-9]*)
                                ;;
                        *)
                                echo "invalid port ${port}" >&2
                                printdef "Invalid port ${port}"
                        ;;
                esac
                ;;
        -dn)
                domain="$2"
                shift 2

                if ! printf "%s" "${domain}" | grep -E '^([a-z0-9_]+(-[a-z0-9_]+)*\.)+[a-z0-9_]+([a-z0-9_-]+)$' > /dev/null; then
                        echo "Invalid domain name ${domain}" >&2
                        printdef "Invalid domain name ${domain}"
                fi
                ;;
        -mode)
                mode="$2"
                shift 2
                ;;
        -maxusers)
                maxusers="$2"
                shift 2
                ;;
        *)
                printdef "Unknown option: $1"
                ;;
    esac
done

if [ -z "$mode" ]; then
        mode=$MODE_BRIGADE
fi

if [ -z "$maxusers" ]; then
        maxusers=$DEFAULT_MAXUSERS
fi

if [ "${mode}" != $MODE_VGSOCKET ] && [ "${mode}" != $MODE_BRIGADE ]; then
        printdef "Unknown mode: ${mode}"
fi

if [ -z "$brigade_id" ] \
|| [ -z "$endpoint_ip4" ] || [ -z "$ip4_cgnat" ] || [ -z "$ip6_ula" ] || [ -z "$dns_ip4" ] || [ -z "$dns_ip6" ] \
|| [ -z "$keydesk_ip6" ]; then
        printdef "Not enough arguments"
fi

if [ "${mode}" = $MODE_BRIGADE ]; then
        if [ -z "$brigadier_name" ] || [ -z "$person_name" ] || [ -z "$person_desc" ] || [ -z "$person_url" ]; then
                printdef "Not enough arguments"
        fi
fi

DB_DIR=${DB_DIR:-"/home/${brigade_id}"}

if [ -z "${wg_configs}" ] && [ -z "${ipsec_configs}" ] && [ -z "${ovc_configs}" ] && [ -z "${outline_configs}" ]; then
        wg_configs="-wg native"
fi

if [ -z "${port}" ]; then
        port="0"
fi

# * Check if brigade is exists
if [ -z "${DEBUG}" ] && [ -s "${DB_DIR}/created" ]; then
        echo "Brigade ${brigade_id} already exists" >&2

        fatal "409" "Conflict" "Brigade ${brigade_id} already exists"
fi

if test -f "${DB_DIR}/.maintenance" && test "$(date '+%s')" -lt "$(cat "${DB_DIR}/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(cat "${DB_DIR}/.maintenance")")"
fi

if test -f "/.maintenance" && test "$(date '+%s')" -lt "$(cat "/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(cat /.maintenance)")"
fi

if  [ -z "${DEBUG}" ] && [ ! -d "${ROUTER_SOCKETS_DIR}" ]; then
        install -o root -g "${VGROUTER_GROUP}" -m 0711 -d "${ROUTER_SOCKETS_DIR}" >&2
fi

# Disable doubled brigades.
grep -s "${endpoint_ip4}" "$(dirname "${DB_DIR}")"/*/brigade.json | grep "endpoint_ipv4" | sed 's/\:.*$//' | while IFS= read -r orphan; do
        echo "DEBUG: doubled brigade $orphan" >&2

        orphan_id="$(basename "$(dirname "${orphan}")")"

        if [ -z "${DEBUG}" ]; then
                systemctl --quiet --force stop vgkeydesk@"${orphan_id}".service ||:
                systemctl --quiet disable vgkeydesk@"${orphan_id}".service ||:

                mv -f "${orphan}" "${orphan}.removed" ||:
        else 
                echo "DEBUG: systemctl --quiet --force stop vgkeydesk@${orphan_id}.service" >&2
                echo "DEBUG: systemctl --quiet disable vgkeydesk@${orphan_id}.service" >&2
                echo "DEBUG: mv -f ${orphan} ${orphan}.removed" >&2
        fi
done

# * Create system user
if [ -z "${DEBUG}" ]; then
        {
                useradd -p '*' -G "${VGCERT_GROUP}" -M -s /usr/sbin/nologin -d "${DB_DIR}" "${brigade_id}" >&2
                install -o "${brigade_id}" -g "${brigade_id}" -m 0700 -d "${DB_DIR}" >&2
                install -o "${brigade_id}" -g "${VGSTATS_GROUP}" -m 0710 -d "${STATS_DIR}/${brigade_id}" >&2
                install -o "${brigade_id}" -g "${VGROUTER_GROUP}" -m 2710 -d "${ROUTER_SOCKETS_DIR}/${brigade_id}" >&2
        
        } || fatal "500" "Internal server error" "Can't create brigade ${brigade_id}"
else
        echo "DEBUG: useradd -p '*' -G ${VGCERT_GROUP} -M -s /usr/sbin/nologin -d ${DB_DIR} ${brigade_id}" >&2
        echo "DEBUG: install -o ${brigade_id} -g ${brigade_id} -m 0700 -d ${DB_DIR}" >&2
        echo "DEBUG: install -o ${brigade_id} -g ${VGSTATS_GROUP} -m 0710 -d ${STATS_DIR}/${brigade_id}" >&2
        echo "DEBUG: install -o ${brigade_id} -g ${VGROUTER_GROUP} -m 2710 -d ${ROUTER_SOCKETS_DIR}/${brigade_id}" >&2
fi

EXECUTABLE_DIR="$(realpath "$(dirname "$0")")"

if [ -z "${DEBUG}" ]; then
        # Create json datafile
        # shellcheck disable=SC2086
        sudo -u "${brigade_id}" -g "${brigade_id}" "${BRIGADE_MAKER_APP_PATH}" \
                -ep4 "${endpoint_ip4}" \
                -dns4 "${dns_ip4}" \
                -dns6 "${dns_ip6}" \
                -int4 "${ip4_cgnat}" \
                -int6 "${ip6_ula}" \
                -kd6 "${keydesk_ip6}" \
                -p "$port" \
                -dn "$domain" \
                ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs} \
                -mode "${mode}" \
                -maxusers "${maxusers}" \
                >&2 || fatal "500" "Internal server error" "Can't create brigade ${brigade_id}"
else
        BRIGADE_SOURCE_DIR="$(realpath "${EXECUTABLE_DIR}")"
        if [ -x "${BRIGADE_MAKER_APP_PATH}" ]; then
                # shellcheck disable=SC2086
                "${BRIGADE_MAKER_APP_PATH}" \
                        -ep4 "${endpoint_ip4}" \
                        -dns4 "${dns_ip4}" \
                        -dns6 "${dns_ip6}" \
                        -int4 "${ip4_cgnat}" \
                        -int6 "${ip6_ula}" \
                        -kd6 "${keydesk_ip6}" \
                        -p "$port" \
                        -dn "$domain" \
                        -id "${brigade_id}" \
                        -d "${DB_DIR}" \
                        -c "${CONF_DIR}" \
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs} \
                        ${apiaddr} \
                        -mode "${mode}" \
                        -maxusers "${maxusers}" \
                        >&2 || fatal "500" "Internal server error" "Can't create brigade ${brigade_id}"
        elif [ -s "${BRIGADE_SOURCE_DIR}/main.go" ]; then
                # shellcheck disable=SC2086
                go run "${BRIGADE_SOURCE_DIR}" \
                        -ep4 "${endpoint_ip4}" \
                        -dns4 "${dns_ip4}" \
                        -dns6 "${dns_ip6}" \
                        -int4 "${ip4_cgnat}" \
                        -int6 "${ip6_ula}" \
                        -kd6 "${keydesk_ip6}" \
                        -p "$port" \
                        -dn "$domain" \
                        -id "${brigade_id}" \
                        -d "${DB_DIR}" \
                        -c "${CONF_DIR}" \
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs} \
                        ${apiaddr} \
                        -mode "${mode}" \
                        -maxusers "${maxusers}" \
                        >&2 || fatal "500" "Internal server error" "Can't create brigade ${brigade_id}"
        else
                echo "ERROR: Can't find ${BRIGADE_MAKER_APP_PATH} or ${BRIGADE_SOURCE_DIR}/main.go" >&2

                fatal "500" "Internal server error" "Can't find create createbrigade binary or source code"
        fi
fi

if [ "${mode}" = $MODE_BRIGADE ]; then
        if [ -z "${DEBUG}" ]; then
                # shellcheck disable=SC2086
                wgconf="$(sudo -u "${brigade_id}" -g "${brigade_id}" "${KEYDESK_APP_PATH}" \
                        -name "${brigadier_name}" \
                        -person "${person_name}" \
                        -desc "${person_desc}" \
                        -url "${person_url}" \
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs} \
                        ${chunked} \
                        ${json} \
                        )" || (echo "$wgconf"; exit 1)
        else
                KEYDESK_SOURCE_DIR="$(realpath "${EXECUTABLE_DIR}/../keydesk")"
                # shellcheck disable=SC2086
                if [ -x "${KEYDESK_APP_PATH}" ]; then
                        wgconf="$("${KEYDESK_APP_PATH}" \
                                -name "${brigadier_name}" \
                                -person "${person_name}" \
                                -desc "${person_desc}" \
                                -url "${person_url}" \
                                -id "${brigade_id}" \
                                -d "${DB_DIR}" \
                                -c "${CONF_DIR}" \
                                ${apiaddr} \
                                ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs} \
                                ${chunked} \
                                ${json} \
                                )" || (echo "$wgconf"; exit 1)
                elif [ -s "${KEYDESK_SOURCE_DIR}/../keydesk/main.go" ]; then
                        # shellcheck disable=SC2086
                        wgconf="$(go run "$(dirname $0 | xargs realpath)/../keydesk" \
                                -name "${brigadier_name}" \
                                -person "${person_name}" \
                                -desc "${person_desc}" \
                                -url "${person_url}" \
                                -id "${brigade_id}" \
                                -d "${DB_DIR}" \
                                -c "${CONF_DIR}" \
                                ${apiaddr} \
                                ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs} \
                                ${chunked} \
                                ${json} \
                                )" || (echo "$wgconf"; exit 1)
                else
                        echo "ERROR: can't find ${KEYDESK_APP_PATH} or ${KEYDESK_SOURCE_DIR}/../keydesk/main.go" >&2

                        fatal "500" "Internal server error" "Can't find keydesk binary or source code"
                fi
        fi
fi

# * Activate keydesk systemD units

systemd_vgkeydesk_instance="vgkeydesk@${brigade_id}"
if [ -z "${DEBUG}" ]; then
        {
                systemctl -q enable "${systemd_vgkeydesk_instance}.service" >&2
                # Start systemD services
                systemctl -q start "${systemd_vgkeydesk_instance}.service" >&2
        } || fatal "500" "Internal server error" "Can't start or enable ${systemd_vgkeydesk_instance}"
else
        echo "DEBUG: systemctl -q enable ${systemd_vgkeydesk_instance}.service" >&2
        echo "DEBUG: systemctl -q start ${systemd_vgkeydesk_instance}.service" >&2
fi

if [ "${mode}" = $MODE_BRIGADE ]; then
        # Print brigadier config
        printf "%s" "${wgconf}"
fi

[ -z "${DEBUG}" ] && date -u +"%Y-%m-%dT%H:%M:%S" > "${DB_DIR}/created"
