# REPLAY 

## Usage

`/opt/vgkeydesk/replay`

* `-b` - replay only brigades and brigadiers creations, don't use with `-u` or `-e` flags
* `-u` - replay only users creations, don't use with `-b` or `-e` or `-r` flags
* `-e` - only delete brigades and users (without creation), don't use with any other mode flags
* `-r` - clean before (with deletion), don't use with `-u` or `-e` flags
* `-id` - (for test only) brigade id (base32 format)
* `-d` - (for test only) directory with brigade files, default is `/home/<BrigadeID>`
* `-a` - (for test only) API endpoint address, `-` - no real API calls, default is not set, address will be calculated




