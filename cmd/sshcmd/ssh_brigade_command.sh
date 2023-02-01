#!/bin/sh

# interpret first argument as command
# pass rest args to scripts

printdef() {
    echo "Usage: <command> <args...>"
    exit 1
}

if [ $# -eq 0 ]; then 
    printdef
fi

cmd=${1}; shift
basedir=$(dirname $0)

if [ "xspawn" = "x${cmd}" -o "xcreate" = "x${cmd}" ]; then
    ${basedir}/create_brigade.sh $@
elif [ "xrecreate" = "x${cmd}" ]; then
    ${basedir}/recreate_brigade.sh $@
elif [ "xdestroy" = "x${cmd}" ]; then
    ${basedir}/destroy_brigade.sh $@
else
    echo "Unknown command: ${cmd}"
    printdef
fi
