#!/bin/sh

set -e

FORCE_INSTALL=$1

# Extract files

INSTALL_DIR="/opt/__install__"
install -g root -o root -m 0500 -d "${INSTALL_DIR}"
awk '/^__PAYLOAD_BEGINS__/ { print NR + 1; exit 0; }' $0 | xargs -I {} tail -n +{} $0 | base64 -d | tar -xzp -C ${INSTALL_DIR} >> /install.log 2>&1


# Install keydesk

if [ "x" != "x${FORCE_INSTALL}" ]; then
    install -g root -o root -m 0555 -d /etc/keydesk
    install -g root -o root -m 0555 -d /opt/keydesk
fi

install -g root -o root -m 005 "${INSTALL_DIR}/bin/keydesk" /opt/keydesk

if [ "x" != "x${FORCE_INSTALL}" ]; then
    install -g root -o root -m 644 "${INSTALL_DIR}/keydesk@.service" /etc/systemd/system
    install -g root -o root -m 644 "${INSTALL_DIR}/keydesk@.socket" /etc/systemd/system
fi

# Cleanup

rm -rf "${INSTALL_DIR}"

exit 0
__PAYLOAD_BEGINS__
