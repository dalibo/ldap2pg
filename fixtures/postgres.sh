#!/bin/bash -eux

# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See openldap-data.ldif for details.

roles=($(psql -tc "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' AND rolname NOT IN (CURRENT_USER, 'postgres');"))
# This is tricky: https://stackoverflow.com/questions/7577052/bash-empty-array-expansion-with-set-u
roles=$(IFS=',' ; echo "${roles[*]+${roles[*]}}")
# Quote rolname for case sensitivity.
roles="${roles//,/'", "'}"

psql="psql -v ON_ERROR_STOP=1 --echo-all"

for d in template1 postgres ; do
    $psql $d <<EOSQL
UPDATE pg_namespace SET nspacl = NULL WHERE nspname NOT LIKE 'pg_%';
GRANT USAGE ON SCHEMA information_schema TO PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO PUBLIC;
DO \$\$BEGIN
  IF '${roles}' <> '' THEN
    DROP OWNED BY "${roles:-pouet}";
  END IF;
END\$\$;
EOSQL
done

$psql <<EOSQL
-- Purge everything.
DROP DATABASE IF EXISTS olddb;
DROP DATABASE IF EXISTS appdb;
DROP DATABASE IF EXISTS nonsuperdb;
DO \$\$BEGIN
  IF '${roles}' <> '' THEN
    DROP ROLE "${roles:-pouet}";
  END IF;
END\$\$;
UPDATE pg_database SET datacl = NULL WHERE datallowconn IS TRUE;
EOSQL

$psql <<'EOSQL'
-- For non-superuser case
CREATE ROLE "nonsuper" LOGIN CREATEROLE;
CREATE DATABASE nonsuperdb WITH OWNER nonsuper;

-- Create role as it should be. for NOOP
CREATE ROLE "ldap_roles" WITH NOLOGIN;
CREATE ROLE "app" WITH NOLOGIN;
CREATE ROLE "daniel" WITH LOGIN;
CREATE ROLE "david" WITH LOGIN;
CREATE ROLE "denis" WITH LOGIN;
CREATE ROLE "alan" WITH SUPERUSER LOGIN IN ROLE ldap_roles;
-- Create alice superuser without login, for ALTER.
CREATE ROLE "ALICE" WITH SUPERUSER NOLOGIN IN ROLE app;

-- Create spurious roles, for DROP.
CREATE ROLE "olivia";
CREATE ROLE "omar" WITH LOGIN;
CREATE ROLE "oscar" WITH LOGIN IN ROLE app;
CREATE ROLE "œdipe";

-- Put them in ldap_roles for drop
GRANT "ldap_roles" to "omar", "olivia", "oscar", "œdipe";

-- Create a role out of scope, for no drop
CREATE ROLE "keepme";
-- kevin is out of ldap, for drop by nonsuper
CREATE ROLE "kevin";

-- Create databases
CREATE DATABASE olddb;
CREATE DATABASE appdb WITH OWNER "app";

-- Revoke connect on group app so that revoke connect from public wont grant it
-- back.
REVOKE CONNECT ON DATABASE "appdb" FROM "app" ;

EOSQL

# Create a legacy table owned by a legacy user. For reassign before drop
# cascade.
PGDATABASE=olddb $psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
ALTER TABLE keepme OWNER TO "oscar";
EOSQL

# grant some privileges to daniel, to be revoked.
PGDATABASE=olddb $psql <<EOSQL
CREATE SCHEMA oldns;
CREATE TABLE oldns.table1 (id SERIAL);
GRANT SELECT ON ALL TABLES IN SCHEMA oldns TO "daniel";

-- For REVOKE
GRANT USAGE ON SCHEMA oldns TO "daniel";
ALTER DEFAULT PRIVILEGES IN SCHEMA oldns GRANT SELECT ON TABLES TO "daniel";
EOSQL

# Ensure daniel has no privileges on appdb, for grant.
PGDATABASE=appdb $psql <<'EOSQL'
CREATE TABLE public.table1 (id SERIAL);
CREATE VIEW public.view1 AS SELECT 'row0';

CREATE SCHEMA appns;
CREATE TABLE appns.table1 (id SERIAL);
CREATE TABLE appns.table2 (id SERIAL);

CREATE FUNCTION appns.func1() RETURNS text AS $$ SELECT 'Coucou!'; $$ LANGUAGE SQL;
CREATE FUNCTION appns.func2() RETURNS text AS $$ SELECT 'Coucou!'; $$ LANGUAGE SQL;

CREATE SCHEMA empty;

-- No grant to olivia.
-- Partial grant for revoke
GRANT SELECT ON TABLE appns.table1 TO "omar";
-- full grant for revoke
GRANT SELECT ON ALL TABLES IN SCHEMA appns TO "oscar";

-- No grant to denis, for first grant.
-- Partial grant for regrant
GRANT SELECT ON TABLE appns.table1 TO "daniel";
-- Full grant for noop
GRANT SELECT ON ALL TABLES IN SCHEMA appns TO "david";
EOSQL

# Setup non-super fixture, independant from usual case.
PGDATABASE=nonsuperdb $psql <<'EOSQL'
REVOKE ALL ON SCHEMA public FROM public;
ALTER SCHEMA public OWNER TO "nonsuper";
ALTER SCHEMA pg_catalog OWNER TO "nonsuper";

-- Create a table owned by kevin, for reassign
CREATE TABLE table0 (id SERIAL);
ALTER TABLE table0 OWNER TO "kevin";

-- Grant for drop owned by
CREATE TABLE table1 (id SERIAL);
ALTER TABLE table1 OWNER TO "nonsuper";
EOSQL

PGDATABASE=nonsuperdb PGUSER=nonsuper $psql <<'EOSQL'
GRANT SELECT ON table1 TO "kevin";
EOSQL
