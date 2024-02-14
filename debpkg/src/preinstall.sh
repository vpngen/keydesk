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
                echo "user ${vgstats_user} already exists"
        else
                if getent group "${vgstats_user}" >/dev/null 2>&1; then
                        useradd -p "*" -g "${vgstats_user}" -M "${vgstats_user}" -s /usr/sbin/nologin -d /nonexistent
                else
                        useradd -p "*" -M "${vgstats_user}" -s /usr/sbin/nologin -d /nonexistent
                fi
        fi

}

upgrade() {
    	printf "\033[32m Pre Install of an upgrade\033[0m\n"
    	# Step 3(upgrade), do what you need
        systemctl stop --all 'vgkeydesk@*.socket' ||:
        
         if [ -f /etc/systemd/system/vgkeydesk@.socket ]; then
                systemctl stop --all 'vgkeydesk@*.socket' ||:
                rm -f /etc/systemd/system/vgkeydesk@.socket

                find /etc/systemd/system/ -path "/etc/systemd/system/vgkeydesk@*.socket.d/*" -type f -name "listen.conf" -delete
                find /etc/systemd/system/ -path "/etc/systemd/system/vgkeydesk@*.socket.d" -type d -empty -delete
        fi
        
        if [ -f /etc/systemd/system/vgstats@.service ]; then
                systemctl stop --all 'vgstats@*.service' ||:
                rm -f /etc/systemd/system/vgstats@.service
        fi

        printf "\033[32m Reload the service unit from disk\033[0m\n"
        systemctl daemon-reload ||:
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


