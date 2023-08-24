#!/bin/bash
set -eux

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)

"${psql[@]}" <<'EOSQL'
CREATE ROLE "ldap_roles";

CREATE ROLE "extra" SUPERUSER;
CREATE DATABASE "extra0" WITH OWNER "extra";
-- For reassign database.
CREATE ROLE "damien" SUPERUSER IN ROLE "ldap_roles";
CREATE DATABASE "extra1" WITH OWNER "damien";

-- Inherit local parent
CREATE ROLE "local_parent" NOLOGIN;

-- Test role config definition.
ALTER ROLE "alter" SET client_min_messages TO 'ERROR';
ALTER ROLE "alter" SET application_name TO 'not-updated';

ALTER ROLE "alizée" SET client_min_messages TO 'NOTICE';
ALTER ROLE "alizée" SET application_name TO 'not-reset';
ALTER ROLE "alizée" CONNECTION LIMIT 5;

CREATE ROLE "nicolas";
ALTER ROLE "nicolas" SET client_min_messages TO 'NOTICE';
ALTER ROLE "nicolas" SET application_name TO 'keep-me';
GRANT "local_parent" TO "nicolas";

CREATE ROLE "domitille with space" IN ROLE "ldap_roles";
EOSQL
