#!/bin/sh

vgcert_group="vgcert"
vgstats_user="vgstats"

cleanInstall() {
	printf "\033[32m Pre Install of an clean install\033[0m\n"
	# Step 3 (clean install), enable the service in the proper way for this platform

        set -e

	if getent group "${vgcert_group}" >/dev/null 2>&1; then
 		echo "group ${vgcert_group} already exists"
	else
		groupadd "${vgcert_group}"
	fi

        if id "${vgstats_user}" >/dev/null 2>&1; then
                echo 'user ${vgstats_user} already exists'
        else
                useradd -p "*" -M "${vgstats_user}" -s /usr/sbin/nologin -d /nonexistent
        fi

}

upgrade() {
    	printf "\033[32m Pre Install of an upgrade\033[0m\n"
    	# Step 3(upgrade), do what you need
        systemctl stop --force 'vgkeydesk@*.socket' 'vgkeydesk@*.service' ||:
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
    printf "\033[31m install... \033[0m\n"
    cleanInstall
    ;;
  "2" | "upgrade")
    printf "\033[31m upgrade... \033[0m\n"
    upgrade
    ;;
  *)
    # $1 == version being installed
    printf "\033[31m default... \033[0m\n"
    cleanInstall
    ;;
esac


