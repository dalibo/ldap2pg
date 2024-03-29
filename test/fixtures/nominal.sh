#!/bin/bash
set -eux

# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See openldap-data.ldif for details.

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)


"${psql[@]}" <<'EOSQL'
CREATE ROLE "ldap2pg" LOGIN CREATEDB CREATEROLE;

CREATE ROLE "nominal";
GRANT "nominal" TO "ldap2pg";

CREATE DATABASE "nominal" WITH OWNER "nominal";

-- Should be NOLOGIN
CREATE ROLE "readers" LOGIN;
CREATE ROLE "owners" NOLOGIN;

-- For alter
CREATE ROLE "alter";
CREATE ROLE "alizée" IN ROLE "owners";  -- Spurious parent.

-- For drop
CREATE ROLE "daniel" WITH LOGIN;
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
