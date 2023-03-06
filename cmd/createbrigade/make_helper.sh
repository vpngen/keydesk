#!/bin/sh

set -e

printdef () {
        >&2 echo "Usage: $0 <brigabe_id_encoded> <endpoint IPv4> <CGNAT IPv4> <IPv6 ULA> <DNS IPv4> <DNS IPv6> <keydesk IPv6> <Brigadier Name> <Person Name> <Person Desc> <Person URL>"
        exit 1
}

if [ -z "${1}" -o -z "${2}" -o -z "${3}" -o -z "${4}" -o -z "${5}" -o -z "${6}" -o -z "${7}" -o -z "${8}" -o -z "${9}" -o -z "${10}" -o -z "${11}" ]; then 
        printdef
fi

brigade_id=$(echo "${1}" | xxd -r -p -l 16 | base32 | tr -d "=")
endpoint_ip4=${2}
ip4_cgnat=${3}
ip6_ula=${4}
dns_ip4=${5}
dns_ip6=${6}
keydesk_ip6=${7}
brigadier_name=$(echo -n "${8}" | base64 )
person_name=$(echo -n "${9}" | base64 )
person_desc=$(echo -n "${10}" | base64 )
person_url=$(echo -n "${11}" | base64 )

echo "${brigade_id} ${endpoint_ip4} ${ip4_cgnat} ${ip6_ula} ${dns_ip4} ${dns_ip6} ${keydesk_ip6} ${brigadier_name} ${person_name} ${person_desc} ${person_url}"
