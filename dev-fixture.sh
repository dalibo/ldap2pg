#!/bin/bash -eux

if [ -f dev-fixture.sql ] ; then
    psql < dev-fixture.sql
fi

PGDATABASE=legacy PGUSER=oscar psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
EOSQL
