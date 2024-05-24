# Build
```shell
docker build -t keydesk .
```
# Run
```shell
docker run --rm keydesk
```
# Example
```text
$ docker run --rm keydesk
current dir: /opt/vgkeydesk
creating brigade
why create brigade code is 1
getting brigade id
brigade id: PEYDHXO3G5G3BFRRLMCAKTLG54
running keydesk
api addr: localhost:80
running go tests
=== RUN   TestClient
=== RUN   TestClient/create_user
    client_test.go:36: 145 Удачливый Сесил
=== RUN   TestClient/get_user
    client_test.go:45: 0183af87-2dca-455a-ad23-db2f0fe8bd17 141 Обещающий Мазер
    client_test.go:45: a4546b10-8172-4bee-ace2-d16f52fb4e81 145 Удачливый Сесил
--- PASS: TestClient (0.03s)
    --- PASS: TestClient/create_user (0.02s)
    --- PASS: TestClient/get_user (0.00s)
PASS
stopping keydesk, pid 40
```
# Implementation
## Tests
- test code is in [tests](tests) directory
- tests are built with `go test -o /go/bin/ -c github.com/vpngen/keydesk/tests/keydesk` command
- test binaries are called from [test.sh](scripts/test.sh) script, example command `./keydesk.test -test.v -test.run Client -host localhost:80`
## Docker
See [Dockerfile](Dockerfile)
