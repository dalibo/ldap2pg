DROP DATABASE IF EXISTS app0;
DROP DATABASE IF EXISTS app1;
DELETE FROM pg_catalog.pg_auth_members;
DELETE FROM pg_catalog.pg_authid WHERE rolname != 'postgres' AND rolname NOT LIKE 'pg_%';
-- Create role as it should be. for NOOP
CREATE ROLE app0 NOLOGIN;
-- Create a spurious user, for DROP.
CREATE ROLE spurious_group WITH NOLOGIN;
CREATE ROLE spurious WITH LOGIN IN ROLE spurious_group, app0;
-- Create alice superuser without login, for ALTER.
CREATE ROLE alice WITH SUPERUSER NOLOGIN IN ROLE app0;
CREATE DATABASE app0 WITH OWNER app0;
CREATE DATABASE app1;
