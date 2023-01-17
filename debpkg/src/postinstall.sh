#!/bin/sh

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
        systemctl stop 'keydesk@*.socket' 'keydesk@*.service' ||:
    	systemctl daemon-reload ||:

	DBUSER=$(cat /etc/keydesk/dbuser)
	DBNAME=$(cat /etc/keydesk/dbname)

        sql=$(cat <<EOF
BEGIN;

ALTER TABLE IF EXISTS :"schema_name".brigade
	ADD COLUMN IF NOT EXISTS create_at timestamp without time zone NOT NULL DEFAULT NOW();
UPDATE :"schema_name".brigade SET create_at=(SELECT os_counter_mtime FROM :"schema_name".quota LIMIT 1);

ALTER TABLE IF EXISTS :"schema_name".quota
	ADD COLUMN IF NOT EXISTS os_counter_rx uint8 NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS :"schema_name".quota
	ALTER COLUMN os_counter_rx DROP DEFAULT;
ALTER TABLE IF EXISTS :"schema_name".quota
	ADD COLUMN IF NOT EXISTS os_counter_tx uint8 NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS :"schema_name".quota
	ALTER COLUMN os_counter_tx DROP DEFAULT;
ALTER TABLE IF EXISTS :"schema_name".quota
	DROP COLUMN IF EXISTS os_counter_value;

CREATE TABLE IF NOT EXISTS :"schema_name".keydesk (
        single_row_flag         bool UNIQUE NOT NULL DEFAULT true,
        last_visit              timestamp without time zone DEFAULT NULL,
        CONSTRAINT              single_row_check CHECK (single_row_flag = true)
);

INSERT INTO :"schema_name".keydesk DEFAULT VALUES ON CONFLICT DO NOTHING;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA :"schema_name" TO :"schema_name";

COMMIT;
EOF
)

        for schema in $(sudo -u ${DBUSER} psql -d ${DBNAME} -t -c 'SELECT schema_name FROM information_schema.schemata;'); do
                if [ "${schema}" = "public" -o "${schema}" = "pg_catalog" -o "${schema}" = "pg_toast" -o "${schema}" = "information_schema" -o "${schema}" = "_v" -o "${schema}" = "meta" ]; then
                        continue
                fi

                echo "found: ${schema}"

                echo "${sql}" | sudo -u ${DBUSER} psql -d ${DBNAME} -v ON_ERROR_STOP=yes --set schema_name="${schema}"
                rc=$?
                if [ ${rc} -ne 0 ]; then
                        exit 1
                fi
        done

	systemctl start --all 'keydesk@*.socket' ||:
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


