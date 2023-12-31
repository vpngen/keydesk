#!/bin/sh

# interpret first argument as command
# pass rest args to scripts

# This USER must be in the vgstats group

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

if [ "${cmd}" = "fetchstats" ]; then
        "${basedir}/fetchstats" "$@"
else
    echo "Unknown command: ${cmd}"
    printdef
fi
