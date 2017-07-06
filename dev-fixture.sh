#!/bin/bash -eux

if [ -f dev-fixture.sql ] ; then
    psql < dev-fixture.sql
fi

# Create a legacy table owned by a legacy user.
PGDATABASE=legacy PGUSER=oscar psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
EOSQL

# Grant an ACL to one role
psql <<'EOSQL'
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT datname FROM pg_catalog.pg_database WHERE datallowconn
    LOOP
        EXECUTE 'GRANT CONNECT ON DATABASE ' || quote_ident(r.datname) || ' TO alice;';
    END LOOP;
END$$;
EOSQL
