# Keydesk messages
Brigadier notification subsystem.
## Abstract
DC management can communicate with brigadiers via keykesk. 

DC sends messages to brigadier via DC API. API listens to unix socket file, by default `/var/lib/dcapi/<id>/messages.sock`. Endpoints are authorized with JWT tokens.

Brigadier reads messages through http://vpn.works dashboard. Brigadier can sort, filter and paginate messages. Messages are marked as read explicitly. Web dashboard communicates with keydesk via Brigadier API. Brigadier API listens to calculated IPv6 network.

Messages are automatically garbage-collected with the rules:
- if message TTL expired are deleted
- 10 most recent messages with no TTL are saved
- messages with no TTL and older than a month are deleted
- 100 most recent messages are saved
## API
### DC
Documented in [OpenAPI 3 spec](api/messages.yaml).
### Brigadier
Documented in [OpenAPI 2 spec](swagger/swagger.yml).
## Authorization
DC API requires JWT token signed with ECDSA256 private key. Required JWT claims:
- iss: `dc-mgmt`
- aud: `[keydesk]`
- scopes: scopes for each endpoint are documented in DC mgmt API (currently we have only `messages:create` scope)

### Example JWT payload:
```json
{
  "iss": "dc-mgmt",
  "sub": "keydesk",
  "aud": [
    "keydesk"
  ],
  "exp": 1711046285,
  "nbf": 1711042685,
  "iat": 1711042685,
  "jti": "33439698-1d51-4331-92ac-37aed1d2e82e",
  "scopes": [
    "messages:create"
  ]
}
```

There's [CLI utility](cmd/jwt/main.go) for generating tokens for testing.
## Implementation
### Message structure
- id: unique ID of the message, auto generated, unix nanoseconds
- text
- is_read
- priority: optional
- created_at
- ttl: optional
### Storage
Messages are stored in `.messages` field of `brigade.json`.

