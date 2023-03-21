#!/bin/sh

### Delete brigade

# * Stop and disable systemD units
# * Delete system user

# * Remove special brigadier wg-user
# * Remove schema role
# * Remove database schema

BASE_STATS_DIR="/var/lib/vgstats"
BRIGADE_REMOVER_APP_PATH="/opt/vgkeydesk/destroybrigade"

spinlock="`[ ! -z \"${TMPDIR}\" ] && echo -n \"${TMPDIR}/\" || echo -n \"/tmp/\" ; echo \"vgbrigade.spinlock\"`"
trap "rm -f \"${spinlock}\" 2>/dev/null" EXIT
while [ -f "${spinlock}" ] ; do
    sleep 0.1
done
touch "${spinlock}" 2>/dev/null

printdef () {
        echo "Usage: destroy <brigabe_id_encoded>" >&2
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

systemd_vgkeydesk_instance="vgkeydesk@${brigade_id}"
# Stop keydesk systemD services.
systemctl -q -f stop "${systemd_vgkeydesk_instance}.socket" "${systemd_vgkeydesk_instance}.service"
# Disable keydesk systemD srvices.
systemctl -q -f disable "${systemd_vgkeydesk_instance}.socket" "${systemd_vgkeydesk_instance}.service"
# Delete spesial keydesk dir
systemd_vgkeydesk_conf_dir="/etc/systemd/system/${systemd_vgkeydesk_instance}.socket.d"
if [ -d "${systemd_vgkeydesk_conf_dir}" ]; then
        if [ -f "${systemd_vgkeydesk_conf_dir}/listen.conf" ]; then
                rm -f "${systemd_vgkeydesk_conf_dir}/listen.conf"
        fi
        rmdir "${systemd_vgkeydesk_conf_dir}"
fi

systemd_vgstats_instance="vgstats@${brigade_id}"
# Stop stats systemD services.
systemctl -q -f stop "${systemd_vgstats_instance}.service"
# Disable stats systemD srvices.
systemctl -q -f disable "${systemd_vgstats_instance}.service"

# Remove brigade
sudo -u "${brigade_id}" "${BRIGADE_REMOVER_APP_PATH}" -id "${brigade_id}"

if [ -d "${BASE_STATS_DIR}/${brigade_id}" ]; then
        if [ "${BASE_STATS_DIR}/${brigade_id}" != "" -a "${BASE_STATS_DIR}/${brigade_id}" != "/" ]; then
                rm -f "${BASE_STATS_DIR}/${brigade_id}"/*
                rmdir "${BASE_STATS_DIR}/${brigade_id}"
        fi
fi

# Remove system user
userdel -rf "${brigade_id}"
