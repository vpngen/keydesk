#!/bin/sh

### Delete brigade

# * Stop and disable systemD units
# * Delete system user

# * Remove special brigadier wg-user


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
REMOVER_PATH="/opt/vgkeydesk/destroybrigade"
REVIPER_PATH="/opt/vgkeydesk/turnon-vip"
EXECUTABLE_DIR="$(realpath "$(dirname "$0")")"

if [ "root" != "$(whoami)" ]; then
        echo "DEBUG EXECUTION" >&2
        DEBUG="yes"
fi


fatal() {
        fcode="$1"
        fdesc="$2"
        fmsg="$3"

        cat << EOF | awk -v chunked="${chunked}" 'BEGIN {ORS=""} {buf = buf $0 ORS} END {if (chunked != "") print length(buf) "\r\n" buf "\r\n0\r\n\r\n"; else print buf}'
{
        "code": ${fcode},
        "desc": "${fdesc}",
        "status": "error",
        "message": "${fmsg}"
}
EOF
        if [ "${fcode}" = "403" ]; then
                exit 2
        else 
                exit 1
        fi
}

printdef () {
        msg="$1"

        echo "Usage: $0 -id <brigabe_id_encoded>" >&2
        echo "  +debug: -d <db_dir> -s <stats_dir> -a <api_addr>|-" >&2
        
        fatal "400" "Bad request" "${msg}"
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
        -s)
                if [ -z "$DEBUG" ]; then
                        printdef "The -s option is only for debug"
                fi

                STATS_DIR="$2"
                shift 2
                ;;
        -d)
                if [ -z "$DEBUG" ]; then
                        printdef "The -d option is only for debug"
                fi

                DB_DIR="$2"
                shift 2
                ;;
        -a) 
                if [ -z "$DEBUG" ]; then
                        printdef "The -a option is only for debug"
                fi

                apiaddr="-a $2"
                shift 2
                ;;
        -force)
                FORCE="yes"
                shift 1
                ;;
        *)
                printdef "Unknown option: $1"
                ;;
        esac
done


if [ -z "${brigade_id}" ]; then
        printdef "Brigade ID is required"
fi

DB_DIR=${DB_DIR:-"/home/${brigade_id}"}

if [ -z "${FORCE}" ] && test -f "${DB_DIR}/.maintenance" && test "$(date '+%s')" -lt "$(head -n 1 "${DB_DIR}/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(head -n 1 "${DB_DIR}/.maintenance")")"
fi

if [ -z "${FORCE}" ] && test -f "/.maintenance" && test "$(date '+%s')" -lt "$(head -n 1 "/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(head -n 1 /.maintenance)")"
fi

if test -f "${DB_DIR}/.vip" ; then
        if [ -z "${FORCE}" ]; then
                fatal 403 "Forbidden" "Brigade ${brigade_id} is VIP, disable VIP before destroy"
        fi

        if [ -z "${DEBUG}" ]; then
                if id "${brigade_id}" >/dev/null 2>&1; then
                        sudo -u "${brigade_id}" "${REVIPER_PATH}" "-off" >&2 || fatal "500" "Internal server error" "Can't viparize brigade"
                fi
        else
                echo "DEBUG: ${REVIPER_PATH} -off -id ${brigade_id}" >&2
        fi

        rm -f "${DB_DIR}/.vip" || fatal "500" "Internal server error" "Can't remove .vip file"
fi

systemd_vgkeydesk_instance="vgkeydesk@${brigade_id}"
if [ -z "${DEBUG}" ]; then
        {
                # Stop keydesk systemD services.
                systemctl -q -f stop "${systemd_vgkeydesk_instance}.service" ||:
                # Disable keydesk systemD srvices.
                systemctl -q -f disable "${systemd_vgkeydesk_instance}.service" ||:
                # Delete spesial keydesk dir
        } || fatal "500" "Internal server error" "Can't stop or disable ${systemd_vgkeydesk_instance}"
else
        echo "DEBUG: systemctl -q -f stop ${systemd_vgkeydesk_instance}.service" >&2
        echo "DEBUG: systemctl -q -f disable ${systemd_vgkeydesk_instance}.service" >&2
fi

if [ -s "${DB_DIR}/brigade.json" ]; then
        if [ -z "${DEBUG}" ]; then
                # Remove brigade
                # shellcheck disable=SC2086
                if id "${brigade_id}" >/dev/null 2>&1; then
                        sudo -u "${brigade_id}" "${REMOVER_PATH}" -id "${brigade_id}" ${apiaddr} >&2 || fatal "500" "Internal server error" "Can't remove brigade"
                fi
        else
                SOURCE_DIR="$(realpath "${EXECUTABLE_DIR}")"
        
                if [ -x "${REMOVER_PATH}" ]; then
                        # shellcheck disable=SC2086
                        "${REMOVER_PATH}" -id "${brigade_id}" -d "${DB_DIR}" ${apiaddr} >&2 || fatal "500" "Internal server error" "Can't remove brigade"
                elif [ -s "${SOURCE_DIR}/main.go" ]; then
                        # shellcheck disable=SC2086
                        go run "${SOURCE_DIR}/" -id "${brigade_id}" -d "${DB_DIR}" ${apiaddr} >&2 || fatal "500" "Internal server error" "Can't remove brigade"
                else 
                        echo "ERROR: Can't find ${REMOVER_PATH} or ${SOURCE_DIR}/main.go" >&2
        
                        fatal "500" "Internal server error" "Can't find destroy binary or source code"
                fi
        fi
fi

if [ -z "${DEBUG}" ]; then
        {
                if [ -d "${STATS_DIR}/${brigade_id}" ]; then
                        if [ -n "${brigade_id}" ] && [ "${STATS_DIR}/${brigade_id}" != "/" ]; then
                                rm -f "${STATS_DIR}/${brigade_id}"/*
                                rmdir "${STATS_DIR}/${brigade_id}"
                        fi
                fi

                if [ -d "${ROUTER_SOCKETS_DIR}/${brigade_id}" ]; then
                        if [ -n "${brigade_id}" ] && [ "${ROUTER_SOCKETS_DIR}/${brigade_id}" != "/" ]; then
                                rm -f "${ROUTER_SOCKETS_DIR}/${brigade_id}"/*
                                rmdir "${ROUTER_SOCKETS_DIR}/${brigade_id}"
                        fi
                fi
        } || fatal "500" "Internal server error" "Can't remove ${STATS_DIR}/${brigade_id}"
else
        echo "DEBUG: rm -f ${STATS_DIR}/${brigade_id}/*" >&2
        echo "DEBUG: rmdir ${STATS_DIR}/${brigade_id}" >&2
        echo "DEBUG: rm -f ${ROUTER_SOCKETS_DIR}/${brigade_id}/*" >&2
        echo "DEBUG: rmdir ${ROUTER_SOCKETS_DIR}/${brigade_id}" >&2
fi

if [ -z "${DEBUG}" ]; then
        # Remove system user
        if id "${brigade_id}" >/dev/null; then
                userdel -rf "${brigade_id}" || fatal "500" "Internal server error" "Can't remove system user" 
        fi
else 
        echo "DEBUG: userdel -rf ${brigade_id}" >&2
fi
