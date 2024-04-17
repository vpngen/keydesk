#!/bin/bash

#set -e

datadir="/opt/vgkeydesk"
echo "current dir: $(pwd)"

echo "creating brigade"
"$datadir/create_brigade.sh" $("$datadir/brigade-helper") -a - -d . -wg native -ovc amnezia -ipsec text,mobileconfig,ps -outline access_key &> create_brigade.log
echo "why create brigade code is $?" # TODO

echo "getting brigade id"
id="$(jq -r .brigade_id "$datadir/brigade.json")"
echo "brigade id: $id"

echo "running keydesk"
"$datadir/keydesk" -id "$id" -a - -l :80 -jwtpub pub.pem &> keydesk.log &
kdpid=$!

apiaddr="localhost:80"
echo "api addr: $apiaddr"

echo "running go tests"
"$datadir/keydesk.test" -test.v -test.run Client -host localhost:80

echo "stopping keydesk, pid $kdpid"
kill $kdpid
