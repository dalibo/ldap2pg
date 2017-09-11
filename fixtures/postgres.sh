#!/bin/bash -eux

# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See openldap-data.ldif for details.

psql <<EOSQL
-- Purge everything.
DROP DATABASE IF EXISTS legacy;
DROP DATABASE IF EXISTS backend;
DROP DATABASE IF EXISTS frontend;
DELETE FROM pg_catalog.pg_auth_members;
DELETE FROM pg_catalog.pg_authid WHERE rolname != 'postgres' AND rolname NOT LIKE 'pg_%';
REVOKE TEMPORARY ON DATABASE postgres FROM PUBLIC;
REVOKE TEMPORARY ON DATABASE template1 FROM PUBLIC;

-- Create role as it should be. for NOOP
CREATE ROLE backend NOLOGIN;
CREATE ROLE daniel LOGIN;
-- Create spurious roles, for DROP.
CREATE ROLE legacy WITH NOLOGIN;
CREATE ROLE oscar WITH LOGIN IN ROLE legacy, backend;
-- Create alice superuser without login, for ALTER.
CREATE ROLE alice WITH SUPERUSER NOLOGIN IN ROLE backend;

-- Create databases
CREATE DATABASE backend WITH OWNER backend;
REVOKE CONNECT ON DATABASE backend FROM PUBLIC;
CREATE DATABASE frontend;
REVOKE CONNECT ON DATABASE frontend FROM PUBLIC;
CREATE DATABASE legacy;

-- daniel was a backend developer and is now a frontend. He add access to
-- backend database. We have to revoke.
REVOKE CONNECT ON DATABASE frontend FROM daniel;
GRANT CONNECT ON DATABASE backend TO daniel;
EOSQL

# Create a legacy table owned by a legacy user.
PGDATABASE=legacy PGUSER=oscar psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
EOSQL

# grant some privileges to daniel, to be revoked.
PGDATABASE=backend psql <<EOSQL
CREATE SCHEMA backend;
GRANT SELECT ON ALL TABLES IN SCHEMA backend TO daniel;
GRANT USAGE ON SCHEMA backend TO daniel;
ALTER DEFAULT PRIVILEGES IN SCHEMA backend GRANT SELECT ON TABLES TO daniel;
EOSQL

# Ensure daniel has no privileges on frontend, for grant.
PGDATABASE=frontend psql <<EOSQL
CREATE SCHEMA frontend;
CREATE TABLE frontend.table1 (id INTEGER);
CREATE TABLE frontend.table2 (id INTEGER);
CREATE SCHEMA empty;

REVOKE SELECT ON ALL TABLES IN SCHEMA empty FROM daniel;
REVOKE USAGE ON SCHEMA empty FROM daniel;
ALTER DEFAULT PRIVILEGES IN SCHEMA empty REVOKE SELECT ON TABLES FROM daniel;

REVOKE SELECT ON ALL TABLES IN SCHEMA frontend FROM daniel;
REVOKE USAGE ON SCHEMA frontend FROM daniel;
ALTER DEFAULT PRIVILEGES IN SCHEMA frontend REVOKE SELECT ON TABLES FROM daniel;

REVOKE SELECT ON ALL TABLES IN SCHEMA information_schema FROM daniel;
REVOKE USAGE ON SCHEMA information_schema FROM daniel;
ALTER DEFAULT PRIVILEGES IN SCHEMA information_schema REVOKE SELECT ON TABLES FROM daniel;
EOSQL
