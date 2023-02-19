# Brigade management

Lifecycle:
* Create brigade
* Recreate brigadier user (zero or more times)
* Destroy brigade

There is a SSH-based API to manage brigades. The reason is a simple SSH-credentials menegement, simple undestanding and a lot of simple protocol implementations

The special entrypoint user is named `_serega_`. It has sudo permissions to exec entrypoint as a roor system user. 

`_serega_          ALL=(ALL) NOPASSWD: /opt/keydesk/ssh_brigade_command.sh` 

The `authorized_keys` file configuration must force the ssh command:

`command="sudo /opt/keydesk/ssh_brigade_command.sh ${SSH_ORIGINAL_COMMAND}",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty ecdsa-sha2-nistp256 ....`

## Create brigade

* Create the target brigade system user
* Create the target brigade file database and brigade config on the endpoint by execution special script with the target brigade user permissions
* Create the target brigade brigadier user by execution keydesk binary with special flgas with the target brigade user permissions
* Enable and start the target keydesk systemd units
* Enable and start the target brigade stats systemd units

## Destroy brigade

* Stop and disable the target keydesk systemd units
* Stop and disable the target brigade stats systemd units
* Remove target brigade config from endpoint by execution special script with the target brigade user permissions
* Remove the target brigade system user

## Recreate brigadier

* Replace the target brigade brigadier user by execution keydesk binary with special flgas with the target brigade user permissions

## Replay configs

It is an auxilary tool to restore working environment. It replays each command to restore VPN from brigades database.
