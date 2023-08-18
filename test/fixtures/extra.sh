#!/bin/bash
set -eux

# Dév fixture initializing a cluster with a «previous state», needing a lot of
# synchronization. See openldap-data.ldif for details.

export PGUSER=postgres
export PGDATABASE=postgres
psql=(psql -v ON_ERROR_STOP=1 --no-psqlrc)

"${psql[@]}" <<'EOSQL'
CREATE ROLE "extra";
GRANT "extra" TO "ldap2pg";

-- Inherit local parent
CREATE ROLE "local_parent" NOLOGIN;

-- Test role config definition.
ALTER ROLE "alain" SET client_min_messages TO 'ERROR';
ALTER ROLE "alain" SET application_name TO 'not-updated';

ALTER ROLE "alice" SET client_min_messages TO 'NOTICE';
ALTER ROLE "alice" SET application_name TO 'not-reset';
ALTER ROLE "alice" CONNECTION LIMIT 5;

CREATE ROLE "nicolas";
ALTER ROLE "nicolas" SET client_min_messages TO 'NOTICE';
ALTER ROLE "nicolas" SET application_name TO 'keep-me';
GRANT "local_parent" TO "nicolas";

CREATE DATABASE "extra" WITH OWNER "extra";
EOSQL
