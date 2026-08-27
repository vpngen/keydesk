# PRO turn ON/OFF

Mirror of `turnon-vip` for the PRO tier: toggles the `pro` flag in the
brigade storage, which the keydesk then reflects in the `pro` JWT claim.

## Usage

`/opt/vgkeydesk/turnon-pro [options] [-on|-off]`

* `-id` - (for test only) brigade id (base32 format)
* `-d` - (for test only) directory with brigade files, default is `/home/<BrigadeID>`
* `-on` - turn PRO on
* `-off` - turn PRO off

Over SSH the command is exposed as `proon` / `prooff` (see
`cmd/sshcmd/ssh_spawner_command.sh`), same as `vipon` / `vipoff` for VIP.
