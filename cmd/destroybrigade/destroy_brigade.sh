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
REMOVER_PATH="/opt/vgkeydesk/destroybrigade"

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

        echo "Usage: $0 -id <brigabe_id_encoded>" >&2
        echo "  +debug: -d <db_dir> -s <stats_dir> -a <api_addr>|-" >&2
        
        fatal "400" "Bad request" "${msg}"
}

if test "$(date '+%s')" -lt "$(cat ".maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(cat .maintenance)")"
fi
if test "$(date '+%s')" -lt "$(cat "/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(cat /.maintenance)")"
fi

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
        *)
                if [ -n "$NEW_STYLE" ]; then
                        printdef "Unknown option: $1"
                fi

                if [ -z "$1" ]; then 
                        printdef "Brigade ID is required"
                fi

                brigade_id="$1"
                chunked=${2}

                if [ "xchunked" != "x${chunked}" ]; then
                        chunked=""
                else
                        chunked="-ch"
                fi

                break
                ;;
        esac
done


if [ -z "${brigade_id}" ]; then
        printdef "Brigade ID is required"
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

if [ -z "${DEBUG}" ]; then
        # Remove brigade
        # shellcheck disable=SC2086
        if id "${brigade_id}" >/dev/null 2>&1; then
                sudo -u "${brigade_id}" "${REMOVER_PATH}" -id "${brigade_id}" ${apiaddr} >&2 || fatal "500" "Internal server error" "Can't remove brigade"
        fi
else
        DB_DIR=${DB_DIR:-"${STATS_DIR}"}
        EXECUTABLE_DIR="$(realpath "$(dirname "$0")")"
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

if [ -z "${DEBUG}" ]; then
        {
                if [ -d "${STATS_DIR}/${brigade_id}" ]; then
                        if [ -n "${brigade_id}" ] && [ "${STATS_DIR}/${brigade_id}" != "/" ]; then
                                rm -f "${STATS_DIR}/${brigade_id}"/*
                                rmdir "${STATS_DIR}/${brigade_id}"
                        fi
                fi
        } || fatal "500" "Internal server error" "Can't remove ${STATS_DIR}/${brigade_id}"
else
        echo "DEBUG: rm -f ${STATS_DIR}/${brigade_id}/*" >&2
        echo "DEBUG: rmdir ${STATS_DIR}/${brigade_id}" >&2
fi

if [ -z "${DEBUG}" ]; then
        # Remove system user
        if id "${brigade_id}" >/dev/null 2>&1; then
                userdel -rf "${brigade_id}" || fatal "500" "Internal server error" "Can't remove system user" 
        fi
else 
        echo "DEBUG: userdel -rf ${brigade_id}" >&2
fi
