# `pg_dumpacl` [![](https://circleci.com/gh/dalibo/pg_dumpacl.svg?style=shield)](https://circleci.com/gh/dalibo/pg_dumpacl)

A tool to dump ACL per database, based on `pg_dump`

``` console
$ ./pg_dumpacl -l db0
--
-- Database creation
--

CREATE DATABASE "db0" WITH TEMPLATE = template0 OWNER = "postgres";
REVOKE ALL ON DATABASE "db0" FROM PUBLIC;
REVOKE ALL ON DATABASE "db0" FROM "postgres";
GRANT ALL ON DATABASE "db0" TO "postgres";
GRANT CONNECT,TEMPORARY ON DATABASE "db0" TO PUBLIC;
GRANT CONNECT ON DATABASE "db0" TO "dba";
```
