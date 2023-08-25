#!/bin/bash
set -eux

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)

"${psql[@]}" <<EOSQL
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE query IS NOT NULL AND pid <> pg_backend_pid();

DROP DATABASE IF EXISTS nominal;
DROP DATABASE IF EXISTS extra0;
DROP DATABASE IF EXISTS extra1;
DROP DATABASE IF EXISTS big0;
DROP DATABASE IF EXISTS big1;
DROP DATABASE IF EXISTS big2;
DROP DATABASE IF EXISTS big3;
EOSQL

mapfile -t roles < <("${psql[@]}" -Atc "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' AND rolname NOT IN (CURRENT_USER, 'postgres');")
printf -v quoted_roles '"%s", ' "${roles[@]+${roles[@]}}"
quoted_roles="${quoted_roles%, }"

psql=("${psql[@]}" --echo-all)
for role in "${roles[@]+${roles[@]}}" ; do
	"${psql[@]}" <<-EOF
	DROP OWNED BY "${role}" CASCADE;
	EOF
done

"${psql[@]}" <<-EOSQL
GRANT USAGE ON SCHEMA information_schema TO PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO PUBLIC;
GRANT USAGE ON LANGUAGE plpgsql TO PUBLIC;
EOSQL

# Reset default privileges.
"${psql[@]}" -At <<-EOF | "${psql[@]}"
WITH type_map (typechar, typename) AS (
	VALUES
	('T', 'TYPES'),
	('r', 'TABLES'),
	('f', 'FUNCTIONS'),
	('S', 'SEQUENCES')
)
SELECT
	'ALTER DEFAULT PRIVILEGES'
	|| ' FOR ROLE "' ||  pg_get_userbyid(defaclrole) || '"'
	|| ' IN SCHEMA "' || nspname || '"'
	|| ' REVOKE ' || (aclexplode(defaclacl)).privilege_type || ''
	|| ' ON ' || COALESCE(typename, defaclobjtype::TEXT)
	|| ' FROM "' || pg_get_userbyid((aclexplode(defaclacl)).grantee) || '"'
	|| ';' AS "sql"
FROM pg_catalog.pg_default_acl
JOIN pg_namespace AS nsp ON nsp.oid = defaclnamespace
LEFT OUTER JOIN type_map ON typechar = defaclobjtype;
EOF

if [ -n "${roles[*]-}" ] ; then
	"${psql[@]}" <<-EOSQL
	DROP ROLE IF EXISTS ${quoted_roles};
	EOSQL
fi
