#!/bin/bash -eux

if [ -f dev-fixture.sql ] ; then
    psql < dev-fixture.sql
fi

# Create a legacy table owned by a legacy user.
PGDATABASE=legacy PGUSER=oscar psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
EOSQL

# daniel was a backend developer and is now a frontend. He add access to backend
# database. We have to revoke
psql <<EOSQL
GRANT CONNECT ON DATABASE backend TO daniel;
EOSQL
