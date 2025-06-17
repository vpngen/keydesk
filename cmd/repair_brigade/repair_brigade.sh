#!/bin/sh

spinlock="${TMPDIR:-/tmp}/vgbrigade.spinlock"
# shellcheck disable=SC2064
trap "rm -f '${spinlock}' 2>/dev/null" EXIT
while [ -f "${spinlock}" ]; do
    sleep 0.1
done
touch "${spinlock}" 2>/dev/null

#DEBUG=yes

set -e

if [ "$1" = "-r" ]; then 
        REPAIR=yes
        shift
fi

brigade_id=$1
endpoint_ip4=$2
if [ -z "${brigade_id}" ] || [ -z "${endpoint_ip4}" ]; then
        echo "Usage: $0 <brigade_id>"
        echo 1
fi

DB_DIR=${DB_DIR:-"/home/${brigade_id}"}

# * Check if brigade is NOT exists
if [ -z "${DEBUG}" ] && [ ! -s "${DB_DIR}/created" ]; then
        echo "Brigade ${brigade_id} does not exists" >&2

        exit 2
fi

if [ -z "${DEBUG}" ] && [ ! -s "${DB_DIR}/brigade.json" ]; then
        echo "Brigade ${brigade_id} does not exists" >&2

        exit 2
fi

_endpoint_ip4="$(jq -r ".endpoint_ipv4" "${DB_DIR}/brigade.json")"
if [ -z "${endpoint_ip4}" ]; then
        echo "Brigade ${brigade_id} has unexpected ip: ${_endpoint_ip4}, expected ip: ${endpoint_ip4}" >&2

        exit 3
fi

# Disable doubled brigades.
grep -s "${endpoint_ip4}" "$(dirname "${DB_DIR}")"/*/brigade.json | grep "endpoint_ipv4" | sed 's/\:.*$//' | while IFS= read -r orphan; do
        orphan_id="$(basename "$(dirname "${orphan}")")"
        if [ "${orphan_id}" = "${brigade_id}" ]; then
                continue
        fi

        echo "DEBUG: doubled brigade $orphan" >&2

        if [ -z "${DEBUG}" ]; then
                systemctl --quiet --force stop vgkeydesk@"${orphan_id}".service ||:
                systemctl --quiet disable vgkeydesk@"${orphan_id}".service ||:

                mv -f "${orphan}" "${orphan}.removed" ||:

                if [ -n "${REPAIR}" ]; then
                        sudo -u "${brigade_id}" /opt/vgkeydesk/replay -r
                fi
        else
                echo "DEBUG: systemctl --quiet --force stop vgkeydesk@${orphan_id}.service" >&2
                echo "DEBUG: systemctl --quiet disable vgkeydesk@${orphan_id}.service" >&2
                echo "DEBUG: mv -f ${orphan} ${orphan}.removed" >&2
                if [ -n "${REPAIR}" ]; then
                        echo "DEBUG: sudo -u ${brigade_id} /opt/vgkeydesk/replay -r"
                fi
        fi
done