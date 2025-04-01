#!/bin/sh

while [ "$#" -gt 0 ]; do
    case "$1" in
        -d) DATA_DIR="$2";shift 2;;
        -id) BRIGADE_ID="$2"; shift 2;;
        -a) ADDRESS="$2"; shift 2;;
        -no-random) NO_RANDOM=yes; shift 1;;
        -b) BRIGADE_ONLY="-b"; shift 1;;
        -u) USER_ONLY="-u"; shift 1;;
        -r) DELETE_BEFORE_REPLAY="-r"; shift 1;;
        -e) ERASE_ONLY="-e"; shift 1;;
        -do) DELAYED_ONLY="-do"; shift 1;;
        -nd) NO_DELAYED="-nd"; shift 1;;
        *) echo "Unknown parameter: $1"; exit 1;;
    esac
done

DATA_DIR=${DATA_DIR:-"$(dirname "$0")/"}
BRIGADE_ID=${BRIGADE_ID:-"ZBWGQAVTFBFHDIKV4QIB5TZKNM"}
ADDRESS=${ADDRESS:-"-"}

if [ -z "$NO_RANDOM" ]; then
    VGSTATS_RANDOM_DATA=yes
    export VGSTATS_RANDOM_DATA
fi

echo "go run ./$(dirname "$0")/../replay/ -a ${ADDRESS} -d ${DATA_DIR} -id ${BRIGADE_ID} ${BRIGADE_ONLY} ${USER_ONLY} ${DELETE_BEFORE_REPLAY} ${ERASE_ONLY} ${DELAYED_ONLY} ${NO_DELAYED}"
go run ./"$(dirname "$0")"/../replay/ -a "${ADDRESS}" -d "${DATA_DIR}" -id "${BRIGADE_ID}" ${BRIGADE_ONLY} ${USER_ONLY} ${DELETE_BEFORE_REPLAY} ${ERASE_ONLY} ${DELAYED_ONLY} ${NO_DELAYED}
