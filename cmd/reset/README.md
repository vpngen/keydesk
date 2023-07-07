# RESET

## Usage

`/opt/vgkeydesk/reset`

* `-p` - reset endpoint port, -1 - no reset, 0 - random port, default is -1
* `-dn` - reset endpoint domain name, "." - no reset, default is "."
* `-id` - (for test only) brigade id (base32 format)
* `-d` - (for test only) directory with brigade files, default is `/home/<BrigadeID>`
* `-a` - (for test only) API endpoint address, `-` - no real API calls, default is not set, address will be calculated
