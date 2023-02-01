#!/bin/sh

### Delete brigade

# * Stop and disable systemD units
# * Delete system user

# * ? Down keydesk IPv6-address and routes

# * Remove special brigadier wg-user
# * Remove schema role
# * Remove database schema

KEYDESK_CONF_DIR="/etc/keydesk"

# creating brigade and brigadier app.
BRIGADE_REMOVER_APP_PATH="/opt/spawner/destroy"
BRIGADE_MAKER_APP_USER=`cat "${KEYDESK_CONF_DIR}/dbuser" 2>/dev/null`
BRIGADE_MAKER_APP_USER="${BRIGADE_MAKER_APP_USER:-nobody}"

#set -e

printdef () {
        echo "Usage: destroy <brigabe_id_encoded>"
        exit 1
}

if [ -z "$1" ]; then 
        printdef
fi

brigade_id="$1"
chunked=${2}

if [ "xchunked" != "x${chunked}" ]; then
        chunked=""
else
        chunked="-ch"
fi

systemd_instance="keydesk@${brigade_id}"

# Stop systemD services.
systemctl -q -f stop "${systemd_instance}.socket" "${systemd_instance}.service"

# Disable systemD srvices.
systemctl -q -f disable "${systemd_instance}.socket" "${systemd_instance}.service"

# Delete spesial dir
systemd_conf_dir="/etc/systemd/system/${systemd_instance}.socket.d"
if [ -d "${systemd_conf_dir}" ]; then
        if [ -f "${systemd_conf_dir}/listen.conf" ]; then
                rm -f "${systemd_conf_dir}/listen.conf"
        fi
        rmdir "${systemd_conf_dir}"
fi

# Remove system user
userdel -rf ${brigade_id}

# * ? Down keydesk IPv6-address and routes

# * !!! Delete special brigadier wg-user
# * !!! Delete brigade
# * !!! Remove privileges
# * !!! Remove database schema
# * !!! Remove schema role

sudo -i -u ${BRIGADE_MAKER_APP_USER} ${BRIGADE_REMOVER_APP_PATH} ${chunked} -id "${brigade_id}"
