#!/bin/sh


VGROUTER_GROUP="vgrouter"
ROUTER_SOCKETS_DIR="/var/lib/dcapi"

cleanInstall() {
	printf "\033[32m Post Install of an clean install\033[0m\n"
	# Step 3 (clean install), enable the service in the proper way for this platform

        set -e

    	printf "\033[32m Reload the service unit from disk\033[0m\n"
    	systemctl daemon-reload ||:
}

upgrade() {
    	printf "\033[32m Post Install of an upgrade\033[0m\n"
    	# Step 3(upgrade), do what you need
    	printf "\033[32m Reload the service unit from disk\033[0m\n"

        if [ ! -d "${ROUTER_SOCKETS_DIR}" ]; then
                install -o root -g "${VGROUTER_GROUP}" -m 0711 -d "${ROUTER_SOCKETS_DIR}" >&2
        fi

        for b in $(find /home -maxdepth 2 -type f -name created | awk -F'/' '{print $(NF-1)}'); do
                if [ ! -d  "/var/lib/dcapi/$b" ]; then
                        install -o "${b}" -g "${VGROUTER_GROUP}" -m 2710 -d "${ROUTER_SOCKETS_DIR}/${b}" >&2
                fi
        done

        systemctl daemon-reload ||:

        if [ -f /etc/systemd/system/vgkeydesk@.socket ]; then
                systemctl stop --all 'vgkeydesk@*.socket' ||:
                systemctl disable --all 'vgkeydesk@*.socket' ||:

                rm -f /etc/systemd/system/vgstats@.socket
        fi

        if [ -f /etc/systemd/system/vgstats@.service ]; then
                systemctl stop --all 'vgstats@*.service' ||:
                systemctl disable --all 'vgstats@*.service' ||:                

                rm -f /etc/systemd/system/vgstats@.service
        fi

        systemctl daemon-reload ||:
        systemctl restart --all 'vgkeydesk@*.service' ||:
}

# Step 2, check if this is a clean install or an upgrade
action="$1"
if  [ "$1" = "configure" ] && [ -z "$2" ]; then
 	# Alpine linux does not pass args, and deb passes $1=configure
 	action="install"
elif [ "$1" = "configure" ] && [ -n "$2" ]; then
   	# deb passes $1=configure $2=<current version>
	action="upgrade"
fi

case "$action" in
  "1" | "install")
    cleanInstall
    ;;
  "2" | "upgrade")
    printf "\033[32m Post Install of an upgrade\033[0m\n"
    upgrade
    ;;
  *)
    # $1 == version being installed
    printf "\033[32m Alpine\033[0m"
    cleanInstall
    ;;
esac


