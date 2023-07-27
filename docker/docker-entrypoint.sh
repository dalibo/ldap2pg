#!/bin/bash
#
# Bootstrap script for running ldap2pg in container.
#
# This script is mostly stolen from Docker official Postgres image.
#

set -eu
shopt -s nullglob

main() {
	file_env PGPASSWORD
	file_env LDAPPASSWORD

	# check dir permissions to reduce likelihood of half-initialized database
	ls /docker-entrypoint.d/ > /dev/null
	docker_process_init_files /docker-entrypoint.d/*

	echo
	echo "$0: ldap2pg init process complete; ready to start up."
	echo

	exec ldap2pg "$@"
}

# usage: docker_process_init_files [file [file [...]]]
# process initializer files, based on file extensions and permissions
docker_process_init_files() {
	local f
	for f ; do
		case "$f" in
			*.sh)
				if [ -x "$f" ] ; then
					echo "$0: running $f"
					"$f"
				else
					echo "$0: sourcing $f"
					# shellcheck source=/dev/null
					. "$f"
				fi
				;;
			*)
				echo "$0: ignoring $f"
				;;
		esac
	done
}

# usage: file_env VAR [DEFAULT]
#    ie: file_env 'XYZ_DB_PASSWORD' 'example'
# (will allow for "$XYZ_DB_PASSWORD_FILE" to fill in the value of
#  "$XYZ_DB_PASSWORD" from a file, especially for Docker's secrets feature)
file_env() {
	local var="$1"
	local fileVar="${var}_FILE"
	local def="${2:-}"
	if [ "${!var:-}" ] && [ "${!fileVar:-}" ]; then
		echo >&2 "error: both $var and $fileVar are set (but are exclusive)"
		exit 1
	fi
	local val="$def"
	if [ "${!var:-}" ]; then
		val="${!var}"
	elif [ "${!fileVar:-}" ]; then
		val="$(< "${!fileVar}")"
	fi
	export "$var"="$val"
	unset "$fileVar"
}

main "$@"
