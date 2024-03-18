#!/bin/bash
set -eux

# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See openldap-data.ldif for details.

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)


"${psql[@]}" <<'EOSQL'
CREATE ROLE "ldap2pg" LOGIN CREATEDB CREATEROLE;

CREATE ROLE "nominal" ADMIN "ldap2pg";

CREATE DATABASE "nominal" WITH OWNER "nominal";

-- Should be NOLOGIN
CREATE ROLE "readers" LOGIN ADMIN "ldap2pg";
CREATE ROLE "owners" NOLOGIN ADMIN "ldap2pg";

-- For alter
CREATE ROLE "alter" ADMIN "ldap2pg";
CREATE ROLE "alizée" ADMIN "ldap2pg";  -- Spurious parent.

-- For drop
CREATE ROLE "daniel" WITH LOGIN ADMIN "ldap2pg";
EOSQL

version=$("${psql[@]}" -Atc "SELECT current_setting('server_version_num')")
if [ "$version" -lt 160000 ]; then
	"${psql[@]}" <<-'EOSQL'
	ALTER ROLE ldap2pg SUPERUSER;
	EOSQL
fi

# Grantor must be "ldap2pg" to avoid
# WARNING:  role "alizée" has not been granted membership in role "owners" by role "ldap2pg"
# As of Postgres 16.
PGUSER=ldap2pg "${psql[@]}" -d nominal <<'EOSQL'
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
