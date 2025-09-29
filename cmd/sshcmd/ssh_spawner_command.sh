#!/bin/sh

# interpret first argument as command
# pass rest args to scripts

printdef() {
    echo "Usage: <command> <args...>" >&2
    exit 1
}

if [ $# -eq 0 ]; then 
    printdef
fi

cmd=${1}; shift
basedir=$(dirname "$0")

set -e

if [  "${cmd}" = "spawn" ] || [ "create" = "${cmd}" ]; then
    sudo -u root -g root "${basedir}/create_brigade.sh" "$@"
elif [ "${cmd}" = "replace" ]; then
    sudo -u root -g root "${basedir}/replace_brigadier.sh" "$@"
elif [ "${cmd}" = "destroy" ]; then
    sudo -u root -g root "${basedir}/destroy_brigade.sh" "$@"
elif [ "${cmd}" = "vipon" ]; then
    sudo -u root -g root "${basedir}/turnon_vip.sh on"
elif [ "${cmd}" = "vipoff" ]; then
    sudo -u root -g root "${basedir}/turnon_vip.sh off"
else
    echo "Unknown command: ${cmd}"
    printdef
fi
