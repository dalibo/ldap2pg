# Contributing to `ldap2pg`


A `docker-compose.yml` file is provided to setup an OpenLDAP and a PostgreSQL
instances as well as a phpLDAPAdmin to help you manage OpenLDAP.

Setup your environment with regular `PG*` envvars so that `psql` can just
connect to your PostgreSQL instance. `LDAP_HOST`, `LDAP_BIND`, `LDAP_PASSWORD`
and `LDAP_BASE` are used to configure LDAP connection. It's up to you to define
how to access postgres and ldap containers: either use a
`docker-compose.override.yml` to expose port on your host or use docker DNS
resolution.

LDAP admin binddn is `cn=admin,dc=ldap2pg,dc=local` with password `integral`.
`dev-fixture.ldif` provides the data seeding the OpenLDAP.


``` console
$ docker-compose up -d
$ pip install -e .
$ export PGUSER=postgres PGPASSWORD=postgres PGHOST=...
$ export LDAP_BIND=cn=admin,dc=ldap2pg,dc=local LDAP_PASSWORD=integral
$ ldap2pg
```
