#!/bin/sh -eux

dropdb --if-exists db0
dropdb --if-exists db1
dropuser --if-exists alain;
dropuser --if-exists dba;

psql <<EOSQL
CREATE ROLE dba;
CREATE ROLE alain;
GRANT dba TO alain;
EOSQL

createdb db0
psql db0 <<EOSQL
GRANT CONNECT ON DATABASE db0 TO dba;
ALTER DEFAULT PRIVILEGES FOR ROLE alain GRANT ALL PRIVILEGES ON TABLES TO dba;

CREATE SCHEMA s0;
GRANT ALL ON SCHEMA s0 TO dba;
EOSQL

createdb db1
psql db1 <<EOSQL
CREATE SCHEMA s0;
GRANT ALL ON SCHEMA s0 TO dba;

CREATE TABLE s0.t0 (id serial PRIMARY KEY);
EOSQL
