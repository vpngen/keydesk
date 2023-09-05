# OpnVPN over Cloak turn ON

## Usage

`/opt/vgkeydesk/turnon-outline`

* `r` - replay brigade, default is false
* `-p` - purge (need brigadeID) brigade from outline, default is false 
* `-op` - port, 0 - random, default is 0
* `-id` - (for test only) brigade id (base32 format)
* `-d` - (for test only) directory with brigade files, default is `/home/<BrigadeID>`
* `-a` - (for test only) API endpoint address, `-` - no real API calls, default is not set, address will be calculated
