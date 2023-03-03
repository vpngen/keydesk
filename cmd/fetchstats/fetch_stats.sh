#!/bin/sh

BASE_STATS_DIR="/var/lib/vgstats"

set -e

printdef () {
        echo "Usage: $0 <brigabe_id_encoded>" >&2
        exit 1
}

if [ -z "${1}" ]; then 
        printdef
fi

brigade_id=${1}

if [ ! -f "${BASE_STATS_DIR}/${brigade_id}/stats.json" ]; then
        echo "File not found: ${BASE_STATS_DIR}/${brigade_id}/stats.json" >&2
        exit 1
fi

if [ ! -r "${BASE_STATS_DIR}/${brigade_id}/stats.json" ]; then 
        echo "Can't read statistics file: ${BASE_STATS_DIR}/${brigade_id}/stats.json" >&2
        exit 1
fi

cat "${BASE_STATS_DIR}/${brigade_id}/stats.json"

exit 0
