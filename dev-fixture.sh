#!/bin/bash -eux

if [ -f dev-fixture.sql ] ; then
    psql < dev-fixture.sql
fi

PGDATABASE=app0 PGUSER=spurious psql <<EOSQL
CREATE TABLE keepme (id serial PRIMARY KEY);
EOSQL
