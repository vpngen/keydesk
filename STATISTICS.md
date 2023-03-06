# Statistics

This is brigade statistics service

## Main functions

* Collect statistics
* Set and reset VPN-user throttling according to statistics

## Details

There is a one brigade statistic service (later STATS) to the one brigade. There is a one special system user (later USER) for each brigade. The USER username is the brigade <BrigadeID>. The  service unit are systemd templates with name _vgstats_. The USER username (<BrigadeID>) is a instant name (parameter) of the service. The _vgstats_ service runs stats binary in service mode with the USER permissions. 

The STATS uses a main brigade file database `/home/<BrigadeID>/brigade.json` in a way as a keydesk service. Periodically the STATS reads the brigade database, makes API call to collect VPN raw statistics, merges it with the brigade database, calcs some counters, daily, weekly, monthly, yearly traffic counters. And than save a breef agregated version of statistics in the `/var/lib/vgstats/<BrigadeID>/stats.json` (<BrigadeID>:vgstat 0710). A consumers in the _vgstat_ system group can use this copy.

### USERS AND GROUPS

* `BrigadeID:BrigadeID` - brigade user and group *the user/group pair manages by brigade management process*
* `vgstats` - user for fetch statistics (for fetching statistics) *the user manages by this package*

### FILES

* `/home/<BrigadeID>/brigade.json` - brigade file database
* `/var/lib/vgstats/<BrigadeID>/stats.json` - agregated version of the brigade statics

### BINARIES

* `/opt/vgkeydesk/stats`

### SYSTEMD

* `/etc/systemd/system/vgstats@.service` - _vgstats_ service unit

### CONSTANTS

* Collecting statistics period `DefaultStatisticsFetchingDuration` = 1 minute
* A random delay vefore first start `DefaultJitterValue`           = 10  sec

### API CALLS

* `?stat=<wg_public_key>` - fetch wireguard instance statistics
* `xxx` - set VPN-user throtling on
* `yyy` - reset VPN-user throtling
