#!/bin/sh

### Create brigades

# * Check if brigade already exists
# * Create system user
# * Create homedir

# * Create json datafile
# * Create special brigadier wg-user

# * Activate keydesk systemD units
# * Activate stats systemD units

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

DB_DIR="/home"
STATS_DIR="/var/lib/vgstats"
BRIGADE_MAKER_APP_PATH="/opt/vgkeydesk/createbrigade"
KEYDESK_APP_PATH="/opt/vgkeydesk/keydesk"

VGCERT_GROUP="vgcert"
VGSTATS_GROUP="vgstats"

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
                NEW_STYLE="yes"
                brigade_id="$2"
                shift 2
                ;;
        -ch)
                NEW_STYLE="yes"
                chunked="-ch"
                shift 1
                ;;
        -j)
                NEW_STYLE="yes"
                json="-j"
                shift 1
                ;;
        -d)
                #NEW_STYLE="yes"  -d/-c/-a must be first options
                if [ -z "$DEBUG" ]; then
                        printdef "The '-d' option is only for debug"
                fi

                DB_DIR="$2"
                shift 2
                ;;
        -c)
                #NEW_STYLE="yes"
                if [ -z "$DEBUG" ]; then
                        printdef "The '-c' option is only for debug"
                fi

                CONF_DIR="$2"
                shift 2
                ;;
        -a) 
                #NEW_STYLE="yes"
                if [ -z "$DEBUG" ]; then
                        printdef "The '-a' option is only for debug"
                fi

                apiaddr="-a $2"
                shift 2
                ;;
        -wg)
                NEW_STYLE="yes"
                wg_configs="-wg $2"
                shift 2
                ;;
        -ipsec)
                NEW_STYLE="yes"
                ipsec_configs="-ipsec $2"
                shift 2
                ;;
        -ovc)
                NEW_STYLE="yes"
                ovc_configs="-ovc $2"
                shift 2
                ;;
        -ep4)
                NEW_STYLE="yes"
                endpoint_ip4="$2"
                shift 2
                ;;
        -int4)
                NEW_STYLE="yes"
                ip4_cgnat="$2"
                shift 2
                ;;
        -int6)
                NEW_STYLE="yes"
                ip6_ula="$2"
                shift 2
                ;;
        -dns4)
                NEW_STYLE="yes"
                dns_ip4="$2"
                shift 2
                ;;
        -dns6)
                NEW_STYLE="yes"
                dns_ip6="$2"
                shift 2
                ;;
        -kd6)
                NEW_STYLE="yes"
                keydesk_ip6="$2"
                shift 2
                ;;
        -name)
                NEW_STYLE="yes"
                brigadier_name="$2"
                shift 2
                ;;
        -person)
                NEW_STYLE="yes"
                person_name="$2"
                shift 2
                ;;
        -desc)
                NEW_STYLE="yes"
                person_desc="$2"
                shift 2
                ;;
        -url)
                NEW_STYLE="yes"
                person_url="$2"
                shift 2
                ;;
        -p)
                NEW_STYLE="yes"
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
                NEW_STYLE="yes"
                domain="$2"
                shift 2

                if ! printf "%s" "${domain}" | grep -E '^([a-z0-9_]+(-[a-z0-9_]+)*\.)+[a-z0-9_]+([a-z0-9_-]+)$' > /dev/null; then
                        echo "Invalid domain name ${domain}" >&2
                        printdef "Invalid domain name ${domain}"
                fi
                ;;
        *)
                if [ -n "${NEW_STYLE}" ]; then
                        printdef "Unknown option: $1"
                fi

                if [ -z "${1}" ] || \
                [ -z "${2}" ] || \
                [ -z "${3}" ] || \
                [ -z "${4}" ] || \
                [ -z "${5}" ] || \
                [ -z "${6}" ] || \
                [ -z "${7}" ] || \
                [ -z "${8}" ] || \
                [ -z "${9}" ] || \
                [ -z "${10}" ] || \
                [ -z "${11}" ]; then 
                        printdef "Not enough arguments (old style)"
                fi

                brigade_id=${1}
                endpoint_ip4=${2}
                ip4_cgnat=${3}
                ip6_ula=${4}
                dns_ip4=${5}
                dns_ip6=${6}
                keydesk_ip6=${7}
                brigadier_name=${8}
                person_name=${9}
                person_desc=${10}
                person_url=${11}

                shift 11

                chunked=""
                port="0"
                domain=""

                for i in "$@";
                do
                    case $i in
                        [0-9]*)
                                if [ "$i" -ge 1024 ] && [ "$i" -le 65535 ]; then
                                        port="$i"
                                fi
                                ;;
                        *.*)
                                if printf "%s" "$i" | grep -E '^([a-z0-9_]+(-[a-z0-9_]+)*\.)+[a-z0-9_]+([a-z0-9_-]+)$' > /dev/null; then
                                        domain="$i"
                                fi
                        ;;
                        *)
                                if [ "$i" = "chunked" ]; then
                                        chunked="-ch"
                                fi
                        ;;
                    esac
                done

                break
                ;;
    esac
done

if [ -z "$brigade_id" ] \
|| [ -z "$endpoint_ip4" ] || [ -z "$ip4_cgnat" ] || [ -z "$ip6_ula" ] || [ -z "$dns_ip4" ] || [ -z "$dns_ip6" ] || [ -z "$keydesk_ip6" ] \
|| [ -z "$brigadier_name" ] || [ -z "$person_name" ] || [ -z "$person_desc" ] || [ -z "$person_url" ]; then
        printdef "Not enough arguments"
fi

if [ -z "${wg_configs}" ] && [ -z "${ipsec_configs}" ] && [ -z "${ovc_configs}" ]; then
        wg_configs="-wg native"
fi

if [ -z "${port}" ]; then
        port="0"
fi

# * Check if brigade is exists
if [ -z "${DEBUG}" ] && [ -s "${DB_DIR}/${brigade_id}/created" ]; then
        echo "Brigade ${brigade_id} already exists" >&2
        
        fatal "409" "Conflict" "Brigade ${brigade_id} already exists"
fi

# * Create system user
if [ -z "${DEBUG}" ]; then
        {
                useradd -p '*' -G "${VGCERT_GROUP}" -M -s /usr/sbin/nologin -d "${DB_DIR}/${brigade_id}" "${brigade_id}" >&2
                install -o "${brigade_id}" -g "${brigade_id}" -m 0700 -d "${DB_DIR}/${brigade_id}" >&2
                install -o "${brigade_id}" -g "${VGSTATS_GROUP}" -m 710 -d "${STATS_DIR}/${brigade_id}" >&2
        } || fatal "500" "Internal server error" "Can't create brigade ${brigade_id}"
else 
        echo "DEBUG: useradd -p '*' -G ${VGCERT_GROUP} -M -s /usr/sbin/nologin -d ${DB_DIR}/${brigade_id} ${brigade_id}" >&2
        echo "DEBUG: install -o ${brigade_id} -g ${brigade_id} -m 0700 -d ${DB_DIR}/${brigade_id}" >&2
        echo "DEBUG: install -o ${brigade_id} -g ${VGSTATS_GROUP} -m 710 -d ${STATS_DIR}/${brigade_id}" >&2
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
                ${wg_configs} ${ipsec_configs} ${ovc_configs} \
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
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} \
                        ${apiaddr} \
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
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} \
                        ${apiaddr} \
                        >&2 || fatal "500" "Internal server error" "Can't create brigade ${brigade_id}"
        else 
                echo "ERROR: Can't find ${BRIGADE_MAKER_APP_PATH} or ${BRIGADE_SOURCE_DIR}/main.go" >&2

                fatal "500" "Internal server error" "Can't find create createbrigade binary or source code"
        fi
fi

if [ -z "${DEBUG}" ]; then
# shellcheck disable=SC2086
wgconf="$(sudo -u "${brigade_id}" -g "${brigade_id}" "${KEYDESK_APP_PATH}" \
        -name "${brigadier_name}" \
        -person "${person_name}" \
        -desc "${person_desc}" \
        -url "${person_url}" \
        ${wg_configs} ${ipsec_configs} ${ovc_configs} \
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
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} \
                        ${chunked} \
                        ${json} \
                        )" || (echo "$wgconf"; exit 1)
        elif [ -s "${KEYDESK_SOURCE_DIR}/../keydesk/main.go" ]; then
                # shellcheck disable=SC2086
                wgconf="$(go run "$(dirname $0)/../keydesk/main.go" \
                        -name "${brigadier_name}" \
                        -person "${person_name}" \
                        -desc "${person_desc}" \
                        -url "${person_url}" \
                        -id "${brigade_id}" \
                        -d "${DB_DIR}" \
                        -c "${CONF_DIR}" \
                        ${apiaddr} \
                        ${wg_configs} ${ipsec_configs} ${ovc_configs} \
                        ${chunked} \
                        ${json} \
                        )" || (echo "$wgconf"; exit 1)
        else
                echo "ERROR: can't find ${KEYDESK_APP_PATH} or ${KEYDESK_SOURCE_DIR}/../keydesk/main.go" >&2

                fatal "500" "Internal server error" "Can't find keydesk binary or source code"
        fi
fi

# * Activate keydesk systemD units

systemd_vgkeydesk_instance="vgkeydesk@${brigade_id}"
# create dir for custom config
# https://www.freedesktop.org/software/systemd/man/systemd.unit.html
systemd_vgkeydesk_conf_dir="/etc/systemd/system/${systemd_vgkeydesk_instance}.socket.d"

if [ -z "${DEBUG}" ]; then
        #shellcheck disable=SC2174
        mkdir -p "${systemd_vgkeydesk_conf_dir}" -m 0755 >&2 || fatal "500" "Internal server error" "Can't create ${systemd_vgkeydesk_conf_dir}"
else
        echo "DEBUG: mkdir -p ${systemd_vgkeydesk_conf_dir} -m 0755" >&2
fi

# it;s necessary to listen certain IP

# calculated listen IPv6 
listen_ip6=$(echo "${endpoint_ip4}" | sed 's/\./\n/g' | xargs printf 'fdcc:%02x%02x:%02x%02x::2' | sed 's/:0000/:/g' | sed 's/:00/:/g')

if [ -z "${DEBUG}" ]; then
        {
                cat << EOF > "${systemd_vgkeydesk_conf_dir}/listen.conf"
[Socket]
ListenStream = [${listen_ip6}]:80
ListenStream = [${listen_ip6}]:443
EOF
                systemctl -q enable "${systemd_vgkeydesk_instance}.socket" "${systemd_vgkeydesk_instance}.service" >&2
                # Start systemD services
                systemctl -q start "${systemd_vgkeydesk_instance}.socket" "${systemd_vgkeydesk_instance}.service" >&2
        } || fatal "500" "Internal server error" "Can't start or enable ${systemd_vgkeydesk_instance}"
else
        echo "DEBUG: systemctl -q enable ${systemd_vgkeydesk_instance}.socket ${systemd_vgkeydesk_instance}.service" >&2
        echo "DEBUG: systemctl -q start ${systemd_vgkeydesk_instance}.socket ${systemd_vgkeydesk_instance}.service" >&2
fi

# * Activate stats systemD units

systemd_vgstats_instance="vgstats@${brigade_id}"
if [ -z "${DEBUG}" ]; then
        {
                systemctl -q enable "${systemd_vgstats_instance}.service" >&2
                systemctl -q start "${systemd_vgstats_instance}.service" >&2
        } || fatal "500" "Internal server error" "Can't start or enable ${systemd_vgstats_instance}"
else
        echo "DEBUG: systemctl -q enable ${systemd_vgstats_instance}.service" >&2
        echo "DEBUG: systemctl -q start ${systemd_vgstats_instance}.service" >&2
fi

# Print brigadier config
printf "%s" "${wgconf}"

[ -z "${DEBUG}" ] && date -u +"%Y-%m-%dT%H:%M:%S" > "${DB_DIR}/${brigade_id}/created"
