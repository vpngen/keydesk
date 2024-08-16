#!/bin/sh

while [ "$#" -gt 0 ]; do
    case "$1" in
        -d) DATA_DIR="$2";shift 2;;
        -c) CONFIG_DIR="$2"; shift 2;;
        -id) BRIGADE_ID="$2"; shift 2;;
        -l) LISTEN_ADDR="$2"; shift 2;;
        -no-cors) NO_CORS=yes; shift 1;;
        -a) ADDRESS="$2"; shift 2;;
        -w) WEB_DIR="$2"; shift 2;;
        -m) LISTEN_MESSAGE="$2"; shift 2;;
        -no-random) NO_RANDOM=yes; shift 1;;
        *) echo "Unknown parameter: $1"; exit 1;;
    esac
done

DATA_DIR=${DATA_DIR:-"$(dirname "$0")"}
CONFIG_DIR=${CONFIG_DIR:-"$(dirname "$0")"}
WEB_DIR=${WEB_DIR:-"$(dirname "$0")/../../dist/"}
BRIGADE_ID=${BRIGADE_ID:-"ZBWGQAVTFBFHDIKV4QIB5TZKNM"}
LISTEN_ADDR=${LISTEN_ADDR:-"127.0.0.1:8080"}
ADDRESS=${ADDRESS:-"-"}
LISTEN_MESSAGE=${LISTEN_MESSAGE:-"-"}
if [ -n "$NO_CORS" ]; then
    CORS=""
else
    CORS="-cors"
fi

if [ -z "$NO_RANDOM" ]; then
    VGSTATS_RANDOM_DATA=yes
    export VGSTATS_RANDOM_DATA
fi

go run ./"$(dirname "$0")" -shuffler -m "${LISTEN_MESSAGE}" -a "${ADDRESS}"  -d "${DATA_DIR}" -c "${CONFIG_DIR}" -id "${BRIGADE_ID}" -l "${LISTEN_ADDR}" ${CORS} -w "${WEB_DIR}"