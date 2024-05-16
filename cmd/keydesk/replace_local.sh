#!/bin/sh


while [ "$#" -gt 0 ]; do
    case "$1" in
        -id) BRIGADE_ID="$2"; shift 2;;
        -no-chunked) NO_CHUNKED=yes; shift 1;;
        -d) DATA_DIR="$2"; shift 2;;
        -c) CONF_DIR="$2"; shift 2;;
        -a) ADDRESS="$2"; shift 2;;
        -wg) WG_CONFIG_TYPES="-wg $2"; shift 2;;
        -ipsec) IPSEC_CONFIG_TYPES="-ipsec $2"; shift 2;;
        -ovc) OVC_CONFIG_TYPES="-ovc $2"; shift 2;;
        -outline) OUTLINE_CONFIG_TYPES="-outline $2"; shift 2;;
        *) echo "Unknown option: $1"; exit 1;;
    esac
done

DATA_DIR=${DATA_DIR:-"$(dirname "$0")"}
CONF_DIR=${CONF_DIR:-"$(dirname "$0")"}
ADDRESS=${ADDRESS:-"-"}
BRIGADE_ID=${BRIGADE_ID:-"ZBWGQAVTFBFHDIKV4QIB5TZKNM"}
OUTLINE_CONFIG_TYPES=${OUTLINE_CONFIG_TYPES:-"-outline access_key"}
WG_CONFIG_TYPES=${WG_CONFIG_TYPES:-"-wg native"}
if [ -n "$NO_CHUNKED" ]; then
    CHUNKED=""
else
    CHUNKED="-ch"
fi

# shellcheck disable=SC2086
./"$(dirname "$0")"/../replacebrigadier/replace_brigadier.sh \
        -d "${DATA_DIR}" -c "${CONF_DIR}" \
        -id "${BRIGADE_ID}" \
        -a "${ADDRESS}" \
        ${WG_CONFIG_TYPES} ${IPSEC_CONFIG_TYPES} ${OVC_CONFIG_TYPES} ${OUTLINE_CONFIG_TYPES} \
        ${CHUNKED}