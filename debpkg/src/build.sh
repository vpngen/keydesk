#!/bin/sh

set -e

export CGO_ENABLED=0

BUILDDIR=$(mktemp -d)

cp -f keydesk/systemd/keydesk@.service "${BUILDDIR}"
cp -f keydesk/systemd/keydesk@.socket "${BUILDDIR}"

go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

export GOBIN=${BUILDDIR}

go install github.com/vpngen/keydesk/cmd/keydesk@latest

CWD=$(pwd)
cd ${BUILDDIR}

nfpm package --config /data/nfpm.yaml --target ${BUILDDIR} --packager deb

cd ${CWD}

cp -f ${BUILDDIR}/*.deb /data/pkg
chown ${USER_UID}:${USER_UID} /data/pkg/*.deb
rm -f ${BUILDDIR}/*
rmdir ${BUILDDIR}

