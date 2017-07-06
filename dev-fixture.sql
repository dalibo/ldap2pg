-- Dév fixture initializing a cluster with a «previous state», needing a lot of
-- synchronization. See dev-fixture.ldif for details.

-- Purge everything.
DROP DATABASE IF EXISTS legacy;
DROP DATABASE IF EXISTS backend;
DROP DATABASE IF EXISTS frontend;
DELETE FROM pg_catalog.pg_auth_members;
DELETE FROM pg_catalog.pg_authid WHERE rolname != 'postgres' AND rolname NOT LIKE 'pg_%';

-- Create role as it should be. for NOOP
CREATE ROLE backend NOLOGIN;
CREATE ROLE daniel;
-- Create spurious roles, for DROP.
CREATE ROLE legacy WITH NOLOGIN;
CREATE ROLE oscar WITH LOGIN IN ROLE legacy, backend;
-- Create alice superuser without login, for ALTER.
CREATE ROLE alice WITH SUPERUSER NOLOGIN IN ROLE backend;

-- Create databases
CREATE DATABASE backend WITH OWNER backend;
CREATE DATABASE frontend;
CREATE DATABASE legacy;
