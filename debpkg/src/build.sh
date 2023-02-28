#!/bin/sh

set -e

export CGO_ENABLED=0

RELEASE=${BRANCH:-"latest"}

go install github.com/vpngen/keydesk/cmd/keydesk@${RELEASE}
go install github.com/vpngen/keydesk/cmd/createbrigade@${RELEASE}
go install github.com/vpngen/keydesk/cmd/destroybrigade@${RELEASE}

go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

nfpm package --config "keydesk/debpkg/nfpm.yaml" --target "${SHARED_BASE}/pkg" --packager deb

chown ${USER_UID}:${USER_UID} "${SHARED_BASE}/pkg/"*.deb

