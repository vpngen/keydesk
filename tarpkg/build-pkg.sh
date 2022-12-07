#!/bin/sh

set -e

export CGO_ENABLED=0

go install github.com/vpngen/keydesk/cmd/keydesk@latest

cp keydesk/systemd/keydesk@.service .
cp keydesk/systemd/keydesk@.socket .

rm -f /data/update-keydesk.sh
cp -f keydesk/tarpkg/src/update-keydesk.tpl.sh /data/update-keydesk.sh

tar -zcf - \
        bin/keydesk \
        keydesk@.service \
        keydesk@.socket \
        | base64 >> /data/update-keydesk.sh

chown ${USER_UID}:${USER_UID} /data/update-keydesk.sh
