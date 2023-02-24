# Brigade management

Lifecycle:
* Create brigade
* Recreate brigadier user (zero or more times)
* Destroy brigade

There is a SSH-based API to manage brigades. The reason is a simple SSH-credentials menegement, simple undestanding and a lot of simple protocol implementations

The special entrypoint user is named `_serega_`. It has sudo permissions to exec entrypoint as a roor system user. 

`_serega_          ALL=(ALL) NOPASSWD: /opt/vgkeydesk/ssh_brigade_command.sh` 

The `authorized_keys` file configuration must force the ssh command:

`command="/opt/vgkeydesk/ssh_brigade_command.sh ${SSH_ORIGINAL_COMMAND}",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty ecdsa-sha2-nistp256 ....`

### Users and Groups

* `BrigadeID:BrigadeID` - brigade user and group *the user/group pair manages by brigade management process*
* `vgkeydesk` - each brigade user is in this group (for writing statistics) *the group manages by this package*
* `vgcert` - each brigade user is in this group (for reading TLS crt/key pair) *the group manages by the cert package*
* `vgstat` - user for fetch statistics (for fetching statistics) *the user manages by this package*

## Create brigade

Executes with root permission by `sudo` command

* Exclusive lock on `/tmp/vgbrigade.spinlock`
* Check a special brigades list file `/var/lib/vgkeydesk/brigades.lst` (root:root 0600) if the brigade already exists (collision)
* Create the target brigade system user
* Create the target brigade file database and brigade config on the endpoint by execution special script with the target brigade user permissions
* Create the target brigade brigadier user by execution keydesk binary with special flgas with the target brigade user permissions
* Enable and start the target keydesk systemd units (_vgkeydesk_ service with socket activation)
* Enable and start the target brigade stats systemd unit (_vgstats_ service)
* Add `BrigadeID` to the brigades list file `/var/lib/vgkeydesk/brigades.lst`

## Destroy brigade

Executes with root permission by `sudo` command

* Exclusive lock on `/tmp/vgbrigade.spinlock`
* Do not check a special brigades list file `/var/lib/vgkeydesk/brigades.lst`! It does not matter.
* Stop and disable the target keydesk systemd units (_vgkeydesk_ service with socket activation)
* Stop and disable the target brigade stats systemd unit (_vgstats_ service)
* Remove target brigade config from endpoint by execution special script with the target brigade user permissions
* Remove the target brigade system user
* Remove `BrigadeID` from the brigades list file `/var/lib/vgkeydesk/brigades.lst`

## Recreate brigadier

Executes with root permission by `sudo` command

* Exclusive lock on `/tmp/vgbrigade.spinlock`
* Check a special brigades list file `/var/lib/vgkeydesk/brigades.lst` for brigade presents
* Replace the target brigade brigadier user by execution keydesk binary with special flgas with the target brigade user permissions

## Replay configs

Executes with root permission by `sudo` command

It is an auxilary tool to restore working environment. It replays each command to restore VPN from brigades database.
