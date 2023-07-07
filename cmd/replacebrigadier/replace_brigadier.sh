#!/bin/sh

BASE_HOME_DIR="/home"
KEYDESK_APP_PATH="/opt/vgkeydesk/keydesk"

spinlock="`[ ! -z \"${TMPDIR}\" ] && echo -n \"${TMPDIR}/\" || echo -n \"/tmp/\" ; echo \"vgbrigade.spinlock\"`"
trap "rm -f \"${spinlock}\" 2>/dev/null" EXIT
while [ -f "${spinlock}" ] ; do
    sleep 0.1
done
touch "${spinlock}" 2>/dev/null

set -e

printdef () {
        echo "Usage: $0 <brigabe_id_encoded> [chunked] [json]" >&2
        exit 1
}

if [ -z "${1}" ]; then 
        printdef
fi

brigade_id=${1}
chunked=${2}

if [ "chunked" != "${chunked}" ]; then
        chunked=""
else
        chunked="-ch"
fi

# * Check if brigade does not exists
# !!! lib???
if [ ! -s "${BASE_HOME_DIR}/${brigade_id}/created" ]; then
        echo "Brigade ${brigade_id} does not exists" >&2
        exit 1
fi

wgconf=$(sudo -u "${brigade_id}" -g  "${brigade_id}" "${KEYDESK_APP_PATH}" -r "${chunked}")
rc=$?
if [ $rc -ne 0 ]; then
        exit 1
fi

# Print brigadier config
echo "${wgconf}"
