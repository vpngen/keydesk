#!/bin/sh

set -e

export CGO_ENABLED=0

go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
go install github.com/vpngen/keydesk/cmd/keydesk@latest

nfpm package --config "${SHARED_BASE}/nfpm.yaml" --target "${SHARED_BASE}/pkg" --packager deb

chown ${USER_UID}:${USER_UID} "${SHARED_BASE}/pkg/"*.deb

