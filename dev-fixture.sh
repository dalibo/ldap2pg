#!/bin/bash -eux
# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See dev-fixture.ldif for details.

psql <<EOSQL
-- Purge everything.
DROP DATABASE IF EXISTS legacy;
DROP DATABASE IF EXISTS backend;
DROP DATABASE IF EXISTS frontend;
DELETE FROM pg_catalog.pg_auth_members;
DELETE FROM pg_catalog.pg_authid WHERE rolname != 'postgres' AND rolname NOT LIKE 'pg_%';

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
CREATE DATABASE legacy;

-- daniel was a backend developer and is now a frontend. He add access to
-- backend database. We have to revoke.
GRANT CONNECT ON DATABASE backend TO daniel;
EOSQL

# Create a legacy table owned by a legacy user.
PGDATABASE=legacy PGUSER=oscar psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
EOSQL
