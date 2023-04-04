#!/bin/sh

set -e

export CGO_ENABLED=0

go build -C keydesk/cmd/keydesk -o ../../../bin/keydesk
go build -C keydesk/cmd/stats -o ../../../bin/stats
go build -C keydesk/cmd/createbrigade -o ../../../bin/createbrigade
go build -C keydesk/cmd/replay -o ../../../bin/replay
go build -C keydesk/cmd/destroybrigade -o ../../../bin/destroybrigade
go build -C keydesk/cmd/fetchstats -o ../../../bin/fetchstats

go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

nfpm package --config "keydesk/debpkg/nfpm.yaml" --target "${SHARED_BASE}/pkg" --packager deb

chown ${USER_UID}:${USER_UID} "${SHARED_BASE}/pkg/"*.deb

