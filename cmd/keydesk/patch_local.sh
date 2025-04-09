#!/bin/sh

while [ "$#" -gt 0 ]; do
    case "$1" in
        -d) DATA_DIR="$2";shift 2;;
        -id) BRIGADE_ID="$2"; shift 2;;
        -n) DRYRUN="-n"; shift 1;;
        *) FILENAME="$1"; shift 1;;
    esac
done

if [ ! -r "${FILENAME}" ]; then
    echo "File not found or not readable: ${FILENAME}"
    exit 1
fi

DATA_DIR=${DATA_DIR:-"$(dirname "$0")/"}
BRIGADE_ID=${BRIGADE_ID:-"ZBWGQAVTFBFHDIKV4QIB5TZKNM"}

echo "go run ./$(dirname "$0")/../patch/ -d ${DATA_DIR} -id ${BRIGADE_ID} ${DRYRUN}"
go run ./"$(dirname "$0")"/../patch/ -d "${DATA_DIR}" -id "${BRIGADE_ID}"  ${DRYRUN} "${FILENAME}"
