# Keydesk

This is a main brigadier dashboard

## Main functions

* The brigadier web-dashboard service
* The brigadier VPN-user (re)creation utility

## Details

Each VPN brigade has its unique UUID. base32 encoding (without padding) of the UUID uses as a readable compressed <BrigadeID>. There is a one brigadier web-dashboard service (keydesk) to the one brigade. There is a one special system user (later USER) for each brigade. The USER username is the brigade <BrigadeID>. The keydesk service uses systemd socket activation. The socket unit and relevant service unit are systemd templates with name _vgkeydesk_. The USER username (<BrigadeID>) is a instant name (parameter) of the service. There are two ListenStream directives in the socket unit: the first - for HTTP listening, the second - for HTTPS listening. The _vgkeydesk_ service runs keydesk binary in service mode with the USER permissions. 

A brigade user database is a json-encoded file. It locates in the USER home directory: `/home/<BrigadeID>/brigade.json`. Any proccess which wants to read the database must asquire READ-LOCK on the database file. Any process which wants to edit the database must asquire EXCLUSIVE-LOCK on the database temporary file with suffix `.tmp` and truncate this temporary file or create this file and asquire EXCLUSIVE-LOCK on it. Then the process must asquire EXCLUSIVE-LOCK on the database file itself. All changes must be made in the temporary file. At the end the temporary file syncs, renames itself to the main database file, closes and than releases locks. Then the old database file closes and releases locks (the file disappears as a result).

### Consequence

* The process which destroys a brigade must asquire the brigade temporary database file EXCLUSIVE-LOCK for avoid phantom commands to endpoint API.

### Adrersses 

Keydesk listens IPv6 address which calculates from corresponding external IPv4 brigade address: `fdcc:WWXX:YYZZ::2` Where `WW`, `XX`, `YY`, `ZZ` - are hex formatted parts of the IPv4 address `WW.XX.YY.ZZ`. The endpoint side has similar IPv6 address but with `3` ending.

API calls makes to corresponding endpoint side address

A public keydesk IPv6 address link with the described keydesk address with magick. 

### Users and Groups

* `BrigadeID:BrigadeID` - brigade user and group *the user/group pair manages by brigade management process*
* `vgcert` - each brigade user is in this group (for reading TLS crt/key pair) *the group manages by the cert package*

### FILES

* `/home/<BrigadeID>/brigade.json` - brigade file database
* `/etc/vg-router.json` - this node specific nacl public key
* `/etc/vg-shuffler.json` - this realm specific nacl public key
* `/etc/vgcert/vpn.works.crt`,  `/etc/vgcert/vpn.works.crt` - fullchain and key files (Letsencrypt) to keydesks

### BINARIES

* `/opt/vgkeydesk/keydesk`

### SYSTEMD

* `/etc/systemd/system/vgkeydesk@.service` - _vgkeydesk_ service unit
* `/etc/systemd/system/vgkeydesk@.socket` - _vgkeydesk_ socket unit
* `/etc/systemd/system/vgkeydesk@<BrigadeID>.socket.d/listen.conf` - _vgkeydesk_ custom config with ListenStream directives

### CONSTANTS

* Max users per brigade	`MaxUsers` = 250
* Monthly quota per VPN-user `MonthlyQuotaRemaining` = 100 Gb (in+out)

### API CALLS

* `?peer_add=<wg_peer_public_key>` - add wireguard peer (VPN-user)
* `?peer_del=<wg_peer_public_key>` - delete wireguard peer (VPN-user)

