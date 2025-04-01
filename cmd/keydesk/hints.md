## SOCKET

### Create key

`go run ./../jwt/ -aud keydesk -iss dc-mgmt -key ../../../jwt-priv-messages-stage.pem -scopes "configs:create,configs:delete,configs:block,configs:unblock" -sub keydesk -ttl 1000000h`

`export ACCESS_KEY="xxx"`

### Use curl

`curl --unix-socket shuffler.sock -X POST "http://localhost/configs" -H "Authorization: Bearer ${ACCESS_KEY}" -H "Content-Type: application/json" -d '{"configs":["outline"]}'`

`curl --unix-socket shuffler.sock -X DELETE "http://localhost/configs/fe490a7e-478a-4daf-8a62-438fac05d0b2" -H "Authorization: Bearer ${ACCESS_KEY}"`


`curl --unix-socket shuffler.sock -X PATCH "http://localhost/configs/fe490a7e-478a-4daf-8a62-438fac05d0b2/block" -H "Authorization: Bearer ${ACCESS_KEY}"`

`curl --unix-socket shuffler.sock -X PATCH "http://localhost/configs/fe490a7e-478a-4daf-8a62-438fac05d0b2/unblock" -H "Authorization: Bearer ${ACCESS_KEY}"`

`curl --unix-socket shuffler.sock -X GET "http://localhost/slots" -H "Authorization: Bearer ${ACCESS_KEY}"`

`curl --unix-socket shuffler.sock -X GET "http://localhost/activity" -H "Authorization: Bearer ${ACCESS_KEY}"`