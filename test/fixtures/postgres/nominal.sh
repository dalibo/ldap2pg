#!/bin/bash
set -eux

# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See openldap-data.ldif for details.

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)


"${psql[@]}" <<'EOSQL'
CREATE ROLE "ldap2pg" LOGIN CREATEDB CREATEROLE;
EOSQL

version=$("${psql[@]}" -Atc "SELECT current_setting('server_version_num')")
if [ "$version" -ge 160000 ]; then
	"${psql[@]}" <<-'EOSQL'
	ALTER ROLE "ldap2pg" SET createrole_self_grant TO 'set,inherit';
	EOSQL
else
	"${psql[@]}" <<-'EOSQL'
	ALTER ROLE ldap2pg SUPERUSER;
	EOSQL
fi

PGUSER=ldap2pg

"${psql[@]}" <<'EOSQL'
CREATE ROLE "nominal";

CREATE DATABASE "nominal" WITH OWNER "nominal";

-- Should be NOLOGIN
CREATE ROLE "readers" LOGIN;
CREATE ROLE "owners" NOLOGIN;

-- For alter
CREATE ROLE "alter";
CREATE ROLE "alizée";  -- Spurious parent.

-- For drop
CREATE ROLE "daniel" WITH LOGIN;

GRANT "owners" TO "alizée";
EOSQL

"${psql[@]}" -d nominal <<'EOSQL'
ALTER SCHEMA "public" OWNER TO "nominal";

CREATE SCHEMA "nominal"
AUTHORIZATION "nominal"
CREATE TABLE "t0" (id serial PRIMARY KEY)
CREATE TABLE "t1" (id serial PRIMARY KEY);

-- Partial grant on all tables, for regrant
GRANT SELECT ON TABLE "nominal"."t0" TO "readers";
-- missing grant on t1.

-- For revoke.
GRANT UPDATE ON TABLE "nominal"."t0" TO "readers";
EOSQL
