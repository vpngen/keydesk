#!/bin/sh

### Turn on/off VIP for brigadier

# [ ${FLOCKER} != $0 ] && exec env FLOCKER="$0" flock -e "$0" "$0" $@ ||
spinlock="${TMPDIR:-/tmp}/vgbrigade.spinlock"
# shellcheck disable=SC2064
trap "rm -f '${spinlock}' 2>/dev/null" EXIT
while [ -f "${spinlock}" ]; do
    sleep 0.1
done
touch "${spinlock}" 2>/dev/null

set -e

EXECUTABLE_PATH="/opt/vgkeydesk/turnon-vip"

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
        echo "  +debug: -d <db_dir>" >&2
        
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
        -d)
                if [ -z "$DEBUG" ]; then
                        printdef "The -d option is only for debug"
                fi

                DB_DIR="$2"
                shift 2
                ;;
        -on|-off)
                action="$1"
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

if test -f "${DB_DIR}/.maintenance" && test "$(date '+%s')" -lt "$(head -n 1 "${DB_DIR}/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(head -n 1 "${DB_DIR}/.maintenance")")"
fi

if [ -d "${DB_DIR}" ]; then
        if [ "${action}" = "-on" ];  then
                touch "${DB_DIR}/.vip" || fatal "500" "Internal server error" "Can't create .vip file"
        fi

        if [ -z "${DEBUG}" ]; then
                if id "${brigade_id}" >/dev/null 2>&1; then
                        sudo -u "${brigade_id}" "${EXECUTABLE_PATH}" "${action}" >&2 || fatal "500" "Internal server error" "Can't viparize brigade"
                fi
        else
                SOURCE_DIR="$(realpath "$(realpath "$(dirname "$0")")")"
        
                if [ -x "${EXECUTABLE_PATH}" ]; then
                        # shellcheck disable=SC2086
                        "${EXECUTABLE_PATH}" -id "${brigade_id}" -d "${DB_DIR}" "${action}" >&2 || fatal "500" "Internal server error" "Can't viparize brigade"
                elif [ -s "${SOURCE_DIR}/main.go" ]; then
                        # shellcheck disable=SC2086
                        go run "${SOURCE_DIR}/" -id "${brigade_id}" -d "${DB_DIR}" "${action}" >&2 || fatal "500" "Internal server error" "Can't viparize brigade"
                else 
                        echo "ERROR: Can't find ${EXECUTABLE_PATH} or ${SOURCE_DIR}/main.go" >&2
        
                        fatal "500" "Internal server error" "Can't find binary or source code"
                fi
        fi

        if [ "${action}" = "-off" ]; then
                rm -f "${DB_DIR}/.vip" || fatal "500" "Internal server error" "Can't remove .vip file"
        fi
else
        fatal "404" "Not found" "Brigade ${brigade_id} does not exists"
fi
