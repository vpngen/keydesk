#!/bin/sh

set -e

export CGO_ENABLED=0

go build -C keydesk/cmd/keydesk -o ../../../bin/keydesk
go build -C keydesk/cmd/createbrigade -o ../../../bin/createbrigade
go build -C keydesk/cmd/replay -o ../../../bin/replay
go build -C keydesk/cmd/reset -o ../../../bin/reset
go build -C keydesk/cmd/turnon-ovc -o ../../../bin/turnon-ovc
go build -C keydesk/cmd/turnon-ipsec -o ../../../bin/turnon-ipsec
go build -C keydesk/cmd/turnon-outline -o ../../../bin/turnon-outline
go build -C keydesk/cmd/destroybrigade -o ../../../bin/destroybrigade
go build -C keydesk/cmd/fetchstats -o ../../../bin/fetchstats

go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

nfpm package --config "keydesk/debpkg/nfpm.yaml" --target "${SHARED_BASE}/pkg" --packager deb

chown "${USER_UID}:${USER_UID}" "${SHARED_BASE}/pkg/"*.deb

