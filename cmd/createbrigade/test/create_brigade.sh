#!/bin/sh

printdef() {
    echo "Usage: [-tag <tag>]"
    exit 1
}

random_byte() {
  shuf -i 0-255 -n 1
}

random_block() {
  printf '%04x' "$(shuf -i 0-65535 -n 1)"
}



CONF_DIR=${CONF_DIR:-"../../keydesk"}
CONF_DIR="$(realpath "${CONF_DIR}")"

BRIGADE_MAKER_APP_PATH=${BRIGADE_MAKER_APP_PATH:-"$(realpath "../../createbrigade/createbrigade")"}
KEYDESK_APP_PATH=${KEYDESK_APP_PATH:-"$(realpath "../../keydesk/keydesk")"}

while [ "$#" -gt 0 ]; do
    case "$1" in
        -id)
                id="$2"
                shift 2
                ;;
        -ep4)
                ep4="$2"
                shift 2
                ;;
        -dn)
                dn="$2"
                shift 2
                ;;
        *)
            printdef
            ;;
    esac
done

id=${id:-"$(uuidgen -r | xxd -r -p -l 16 | base32 | tr -d "=")"}

DB_DIR=${DB_DIR:-"../../keydesk/testing/${id}"}
DB_DIR="$(realpath "${DB_DIR}")"
if [ ! -d "${DB_DIR}" ]; then
    mkdir -p "${DB_DIR}"
fi

echo "id: ${id}"

ep4=${ep4:-"$(random_byte).$(random_byte).$(random_byte).$(random_byte)"}
dn=${dn:-"test.$(uuidgen -r).com"}

echo "ep4: ${ep4}"
echo "dn: ${dn}"

dns4="100.64.$(random_byte).$(random_byte)"
int4="${dns4}/24"

echo "dns4: ${dns4}"
echo "int4: ${int4}"

dns6="fd10:$(random_block)::$(random_block)"
int6="${dns6}/64"

echo "dns6: ${dns6}"
echo "int6: ${int6}"

kd6="fd20:$(random_block)::$(random_block)"

echo "kd6: ${kd6}"

names="$(go run ../../../../vpngen-wordsgens/main.go -sh ph)"
name="$(echo "${names}" | head -n 1)"
person="$(echo "${names}" | head -n 2 | tail -n 1 | sed 's/^.*\: //')"
desc="$(echo "${names}" | head -n 3 | tail -n 1)"
url="$(echo "${names}" | head -n 4 | tail -n 1)"

echo "name: ${name}"
echo "person: ${person}"
echo "desc: ${desc}"
echo "url: ${url}"

# -id: <brigabe_id_encoded> 
# -ep4: <endpoint IPv4> 
# -int4: <CGNAT IPv4> 
# -int6: <IPv6 ULA> 
# -dns4: <DNS IPv4> 
# -dns6: <DNS IPv6> 
# -kd6: <keydesk IPv6> 
# -name: <B1rigadier Name :: base64> 
# -person: <Person Name :: base64> 
# -desc: <Person Desc :: base64> 
# -url: <Person URL :: base64>
# [-dn]: <domain name>
# [-port]: <wg port>
# [-oport]: <outline port>
# [-mode]: bridge|vgsocket
# [-maxusers]: <max users>
# [-ch]: chunked
# [-j]: json
# -d: <db_dir>
# -c: <conf_dir>
# -a: <api_addr>

BRIGADE_MAKER_APP_PATH="${BRIGADE_MAKER_APP_PATH}" \
KEYDESK_APP_PATH="${KEYDESK_APP_PATH}" \
../create_brigade.sh -id "${id}" -ep4 "${ep4}" -dn "${dn}" \
        -int4 "${int4}" -int6 "${int6}" -dns4 "${dns4}" -dns6 "${dns6}" -kd6 "${kd6}" \
        -name "$(printf "%s" "${name}" | base64)" \
        -person "$(printf "%s" "${person}" | base64)" \
        -desc "$(printf "%s" "${desc}" | base64)" \
        -url "$(printf "%s" "${url}" | base64)" \
        -d "${DB_DIR}" -c "${CONF_DIR}" -a "-" \
        -wg native -ovc amnezia -outline access_key -proto0 access_key

