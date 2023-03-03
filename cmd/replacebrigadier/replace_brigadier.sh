#!/bin/sh

BRIGADES_LIST_FILE="/var/lib/vgkeydesk/brigades.lst"
KEYDESK_APP_PATH="/opt/vgkeydesk/keydesk"

spinlock="`[ ! -z \"${TMPDIR}\" ] && echo -n \"${TMPDIR}/\" || echo -n \"/tmp/\" ; echo \"vgbrigade.spinlock\"`"
trap "rm -f \"${spinlock}\" 2>/dev/null" EXIT
while [ -f "${spinlock}" ] ; do
    sleep 0.1
done
touch "${spinlock}" 2>/dev/null

set -e

printdef () {
        echo "Usage: $0 <brigabe_id_encoded> <Brigadier Name :: base64> <Person Name :: base64> <Person Desc :: base64> <Person URL :: base64>" >&2
        exit 1
}

if [ -z "${1}" -o -z "${2}" -o -z "${3}" -o -z "${4}" -o -z "${5}" ]; then 
        printdef
fi

brigade_id=${1}
brigadier_name=${2}
person_name=${3}
person_desc=${4}
person_url=${5}
chunked=${6}

if [ "xchunked" != "x${chunked}" ]; then
        chunked=""
else
        chunked="-ch"
fi

# * Check if brigade does not exists
# !!! lib???

if [ -f "${BRIGADES_LIST_FILE}" ]; then
        test=$(grep -o "${brigade_id};" < "${BRIGADES_LIST_FILE}" | tr -d ";")
        if [ "x${brigade_id}" -ne "x${test}" ]; then
                echo "Brigade ${brigade_id} does not exists" >&2
                exit 1
        fi 
fi

if ! sudo -u ${brigade_id} -g  ${brigade_id} ${KEYDESK_APP_PATH} -r ${chunked} -name "${brigadier_name}" -person "${person_name}" -desc "${person_desc}" -url "${person_url}"; then
        exit 1
fi

# Print brigadier config
echo "${wgconf}"
