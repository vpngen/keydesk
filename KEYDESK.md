# Keydesk

It is a main brigadier dashboard

## Main functions

* The brigadier user config (re)creation utility
* The brigadier web-dashboard service

## Details

Each brigade has its unique UUID. base32 encoding (without padding) of the UUID is a readable compressed brigade ID. Brigade works with a system permissions with such system username (from this point `BrigadeID`). The HOME directory is `/home/<BrigadeID>`. One keydesk service to one brigade.

A brigade user database is a json-encoded file in the HOME directory: `/home/<BrigadeID>/brigade.json`. Any proccess which wants to read the database must asquire READ-LOCK on the database file. Any process which wants to edit the database must asquire EXCLUSIVE-LOCK on the file named sush as the database file with suffix `.tmp` and truncate this temporary file or create this file and asquire EXCLUSIVE-LOCK on it. Then the process must asquire EXCLUSIVE-LOCK on the database file itself. All changes makes in the temporary file. At the end temporary file syncs, renames in the database file, temporary file closes its lock releases. Then old database file closes and lock releases (file disappears as a result).

File `/var/lib/vgstat/<BrigadeID>-stats.json` (`/var/db/vgstat` vgstat:brigades 0130) сontains statistics for external usage. The statistic file provides by special brigade related statistic service and consumed by the realm.

### Consequence

* The process which destroys a brigade must asquire the brigade temporary database file EXCLUSIVE-LOCK for avoid phantom commands to endpoint API.

## Service environment

Each keydesk service starts through systemd socket activation. The socket unit and relevant service unit are systemd templates. The system username of the corresponding brigade is a instant name (parameter). The systemd keydesk service works with brigade system user permissions.

### Adrersses 

Keydesk listens IPv6 address which calculates from corresponding external IPv4 brigade address: `fdcc:WWXX:YYZZ::2` Where `WW`, `XX`, `YY`, `ZZ` - are hex formatted parts of the IPv4 address `WW.XX.YY.ZZ`. The endpoint side has similar IPv6 address but with `3` ending.

API calls makes to corresponding endpoint side address

A public keydesk IPv6 address link with the described keydesk address with magick. 
