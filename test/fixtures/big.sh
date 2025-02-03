#!/bin/bash
#
# Initialize a big setup to test performances.
#
set -eux

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)


"${psql[@]}" <<'EOSQL'
CREATE ROLE "bigowner" ADMIN "ldap2pg";

CREATE DATABASE "big0" WITH OWNER "bigowner";
EOSQL

queries=()
for i in {0..255} ; do
	printf -v i "%03d" "$i"
	queries+=("CREATE SCHEMA nsp$i AUTHORIZATION bigowner")
	for j in {0..3} ; do
		printf -v j "%03d" "$j"
		queries+=("CREATE UNLOGGED TABLE nsp$i.t$j (id serial PRIMARY KEY)")
		queries+=("CREATE VIEW nsp$i.v$j AS SELECT * FROM nsp$i.t$j")
	done
	queries+=(";") # End CREATE SCHEMA
	# Randomly create a function
	if (( RANDOM % 2 == 0 )) ; then
		queries+=("CREATE FUNCTION nsp$i.f() RETURNS INTEGER LANGUAGE SQL AS \$\$ SELECT 0 \$\$;")
	fi
done

"${psql[@]}" --echo-queries -d "big0" <<-EOSQL
${queries[*]}
EOSQL

"${psql[@]}" <<-EOF
CREATE DATABASE big1 WITH OWNER "bigowner" TEMPLATE "big0";
CREATE DATABASE big2 WITH OWNER "bigowner" TEMPLATE "big0";
CREATE DATABASE big3 WITH OWNER "bigowner" TEMPLATE "big0";
EOF
