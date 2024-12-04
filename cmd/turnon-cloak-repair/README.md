# OpnVPN over Cloak turn ON

## Usage

`/opt/vgkeydesk/turnon-cloak-repair`

* `r` - replay brigade, default is false
* `-p` - purge (need brigadeID) brigade from cloak, default is false 
* `-id` - (for test only) brigade id (base32 format)
* `-d` - (for test only) directory with brigade files, default is `/home/<BrigadeID>`
* `-a` - (for test only) API endpoint address, `-` - no real API calls, default is not set, address will be calculated
