#!/bin/sh

server=$1

ENCCMD="sed -i 's/\\\"limit_monthly_reset_on\\\"\:\s*\\\"2023\-05\-01T00\:00\:00Z\\\"/\\\"limit_monthly_reset_on\\\"\: \\\"2023\-04\-01T00\:00\:00Z\\\"/g'"
RCOMMAND="-- sh -c \"find /home -maxdepth 2 -mindepth 2 -type f -name brigade.json -print | xargs -L 1 ${ENCCMD} \""

set -x

ssh -o IdentitiesOnly=yes -o IdentityFile=~/.ssh/id_ecdsa.home -o StrictHostKeyChecking=no "${server}" sudo -i "${RCOMMAND}"
rc=$?
if [ $rc -ne 0 ]; then
                echo "[-]         Something wrong: $rc"
fi
