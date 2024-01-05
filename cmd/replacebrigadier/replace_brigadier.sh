#!/bin/sh

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
KEYDESK_PATH="/opt/vgkeydesk/keydesk"

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

if test "$(date '+%s')" -lt "$(cat ".maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(cat .maintenance)")"
fi
if test "$(date '+%s')" -lt "$(cat "/.maintenance")"; then
        fatal 503 "Service is not available" "On maintenance till $(date -d "@$(cat /.maintenance)")"
fi

printdef () {
        msg="$1"

        echo "Usage: $0 -id <brigabe_id_encoded> [-ch] [-j] [-wg <conf types list>] [-ipsec <conf types list>] [-ovc <conf types list>]" >&2
        echo "  +debug: -d <db_dir> -c <conf_dir> -a <api_addr>|-" >&2

        fatal "400" "Bad request" "${msg}"
}

chunked=""
json=""
apiaddr=""

wg_configs=""
ipsec_configs=""
ovc_configs=""
outline_configs=""

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
                if [ -z "$DEBUG" ]; then
                        printdef "The -d option is only for debug"
                fi

                DB_DIR="$2"
                shift 2
                ;;
        -c)
                if [ -z "$DEBUG" ]; then
                        printdef "The -c option is only for debug"
                fi

                CONF_DIR="$2"
                shift 2
                ;;
        -a) 
                if [ -z "$DEBUG" ]; then
                        printdef "The -a option is only for debug"
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
        -outline)
                NEW_STYLE="yes"
                outline_configs="-outline $2"
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

if [ -z "${wg_configs}" ] && [ -z "${ipsec_configs}" ] && [ -z "${ovc_configs}" ] && [ -z "${outline_configs}" ]; then
        wg_configs="-wg native"
fi

# * Check if brigade does not exists
# !!! lib???
if [ -z "${DEBUG}" ] && [ ! -s "${DB_DIR}/${brigade_id}/created" ]; then
        echo "Brigade ${brigade_id} does not exists" >&2
        
        fatal "404" "Not found" "Brigade ${brigade_id} does not exists"
fi

if [ -z "${DEBUG}" ]; then
        # shellcheck disable=SC2086
        output="$(sudo -u "${brigade_id}" -g  "${brigade_id}" "${KEYDESK_PATH}" -r ${json} ${chunked} ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs})" || (echo "$output"; exit 1)
else
        CONF_DIR="${CONF_DIR:-${DB_DIR}}"
        EXECUTABLE_DIR="$(realpath "$(dirname "$0")")"
        SOURCE_DIR="$(realpath "${EXECUTABLE_DIR}/../keydesk")"
        if [ -x "${KEYDESK_PATH}" ]; then
                # shellcheck disable=SC2086
                output="$("${KEYDESK_PATH}" -r -d "${DB_DIR}" -c "${CONF_DIR}" -id "${brigade_id}" ${apiaddr} ${json} ${chunked} ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs})" || (echo "$output"; exit 1)
        elif [ -s "${SOURCE_DIR}/main.go" ]; then
                # shellcheck disable=SC2086
                output="$(go run "${SOURCE_DIR}" -r -d "${DB_DIR}" -c "${CONF_DIR}" -id "${brigade_id}" ${apiaddr} ${json} ${chunked} ${wg_configs} ${ipsec_configs} ${ovc_configs} ${outline_configs})" || (echo "$output"; exit 1)
        else
                echo "ERROR: can't find ${KEYDESK_PATH} or ${SOURCE_DIR}/main.go" >&2
                
                fatal "500" "Internal server error" "Can't find keydesk binary or source code"
        fi
fi

# Print brigadier config
printf "%s" "$output"
