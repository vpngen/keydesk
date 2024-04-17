FROM golang:1.21 as build
WORKDIR /go/src/keydesk

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY gen gen
COPY internal internal
COPY kdlib kdlib
COPY keydesk keydesk
COPY pkg pkg
COPY utils utils
COPY vpnapi vpnapi
COPY tests tests

# binaries
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/keydesk
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/createbrigade
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/destroybrigade
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/brigade-helper
# utils
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/keygen
RUN go build -o /go/bin/ github.com/vpngen/vpngine/nacl
# tests
RUN go test -o /go/bin/ -c github.com/vpngen/keydesk/tests/keydesk

FROM debian:12 as runtime

RUN apt update
RUN apt install -y sudo jq curl

RUN groupadd vgcert
RUN groupadd vgstats
RUN groupadd vgrouter
RUN useradd -m keydesk

RUN mkdir -p /var/lib/dcapi
RUN chown keydesk:keydesk /var/lib/dcapi
RUN mkdir -p /opt/vgkeydesk
RUN chown keydesk:keydesk /opt/vgkeydesk

USER keydesk

WORKDIR /opt/vgkeydesk/

COPY --from=build /go/bin/keydesk .
COPY --from=build /go/bin/createbrigade .
COPY --from=build /go/bin/destroybrigade .
COPY --from=build /go/bin/brigade-helper .
COPY --from=build /go/bin/keygen .
COPY --from=build /go/bin/nacl .
COPY --from=build /go/bin/keydesk.test .
COPY --from=build /go/src/keydesk/cmd/createbrigade/create_brigade.sh .
COPY --from=build /go/src/keydesk/cmd/destroybrigade/destroy_brigade.sh .
COPY --from=build /go/src/keydesk/cmd/replacebrigadier/replace_brigadier.sh .
COPY scripts .

RUN ./nacl genkey > vg-router-private.json
RUN ./nacl pubkey < vg-router-private.json > vg-router.json
RUN ./nacl genkey > vg-shuffler-private.json
RUN ./nacl pubkey < vg-shuffler-private.json > vg-shuffler.json
RUN ./keygen

CMD ["./test.sh"]
