#!/bin/sh

### Create brigades

# * Create system user
# * Create homedir

# * Create json datafile
# * Create special brigadier wg-user

# * Create systemD units

# * Send brigadier config

# creating brigade and brigadier app.

BASE_HOME_DIR="/home"
BRIGADE_MAKER_APP_PATH="/opt/keydesk/create"

set -e

printdef () {
        echo "Usage: $0 <brigabe_id_encoded> <endpoint IPv4> <CGNAT IPv4> <IPv6 ULA> <DNS IPv4> <DNS IPv6> <keydesk IPv6> <Brigadier Name :: base64> <Person Name :: base64> <Person Desc :: base64> <Person URL :: base64>"
        exit 1
}

if [ -z "${1}" -o -z "${2}" -o -z "${3}" -o -z "${4}" -o -z "${5}" -o -z "${6}" -o -z "${7}" -o -z "${8}" -o -z "${9}" -o -z "${10}" -o -z "${11}" ]; then 
        printdef
fi

brigade_id=${1}
endpoint_ip4=${2}
ip4_cgnat=${3}
ip6_ula=${4}
dns_ip4=${5}
dns_ip6=${6}
keydesk_ip6=${7}
brigadier_name=${8}
person_name=${9}
person_desc=${10}
person_url=${11}
chunked=${12}

if [ "xchunked" != "x${chunked}" ]; then
        chunked=""
else
        chunked="-ch"
fi

# * Create system user

useradd -p '*' -M -s /usr/sbin/nologin -d "${BASE_HOME_DIR}/${brigade_id}" "${brigade_id}"

# Create brigadier record
# Create special brigadier wg-user, get brigadier IPv6 and wg.conf

#? create brigade config
#? create brigadier

wgconf=$(sudo -i -u ${brigade_id} ${BRIGADE_MAKER_APP_PATH} ${chunked} -id "${brigade_id}" -ep4 "${endpoint_ip4}" -dns4 "${dns_ip4}" -dns6 "${dns_ip6}" -int4 "${ip4_cgnat}" -int6 "${ip6_ula}" -kd6 "${keydesk_ip6}" -name "${brigadier_name}" -person "${person_name}" -desc "${person_desc}" -url "${person_url}")

# * Create systemD units

systemd_keydesk_instance="keydesk@${brigade_id}"
systemd_stats_instance="stats@${brigade_id}"

# create dir for custom config
# https://www.freedesktop.org/software/systemd/man/systemd.unit.html
systemd_keydesk_conf_dir="/etc/systemd/system/${systemd_keydesk_instance}.socket.d"

mkdir -p "${systemd_keydesk_conf_dir}" -m 0755

# it;s necessary to listen certain IP

# calculated listen IPv6 
listen_ip6=$(echo ${endpoint_ip4} | sed 's/\./\n/g' | xargs printf 'fdcc:%02x%02x:%02x%02x::2' | sed 's/:0000/:/g' | sed 's/:00/:/g')

cat << EOF > "${systemd_keydesk_conf_dir}/listen.conf"
[Socket]
ListenStream = [${listen_ip6}]:80
EOF

systemctl -q enable "${systemd_keydesk_instance}.socket" "${systemd_keydesk_instance}.service"
systemctl -q enable "${systemd_stats_instance}.service"

# Start systemD services
systemctl -q start "${systemd_keydesk_instance}.socket" "${systemd_keydesk_instance}.service"
systemctl -q start "${systemd_stats_instance}.service"

# Print brigadier config
echo "${wgconf}"
