FROM golang:1.21.5-bookworm as build
WORKDIR /go/src/keydesk

ARG gh_user
ARG gh_token
RUN git config --global url."https://${gh_user}:${gh_token}@github.com/".insteadOf "https://github.com/"

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/keydesk
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/createbrigade
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/destroybrigade
RUN go build -o /go/bin/ github.com/vpngen/keydesk/cmd/create-brigade-helper
RUN go build -o /go/bin/ github.com/vpngen/vpngine/nacl

FROM debian:12 as runtime

RUN apt update
RUN apt install -y sudo jq systemctl

RUN groupadd vgcert
RUN groupadd vgstats

WORKDIR /opt/vgkeydesk/

COPY --from=build /go/bin/nacl nacl
RUN ./nacl genkey > vg-router-private.json
RUN ./nacl pubkey < vg-router-private.json > /etc/vg-router.json
RUN ./nacl genkey > vg-shuffler-private.json
RUN ./nacl pubkey < vg-shuffler-private.json > /etc/vg-shuffler.json

COPY --from=build /go/bin/keydesk .
COPY --from=build /go/bin/createbrigade .
COPY --from=build /go/bin/destroybrigade .
COPY --from=build /go/bin/create-brigade-helper .
COPY --from=build /go/src/keydesk/cmd/createbrigade/create_brigade.sh .
COPY --from=build /go/src/keydesk/cmd/destroybrigade/destroy_brigade.sh .
COPY --from=build /go/src/keydesk/cmd/replacebrigadier/replace_brigadier.sh .

