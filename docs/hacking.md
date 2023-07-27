---
hide:
  - navigation
---

<h1>Hacking</h1>

You are welcome to contribute to ldap2pg with patch to code, documentation or
configuration sample ! Here is an extended documentation on how to setup *a*
development environment. Feel free to adapt to your cumfort. Automatic tests on
CircleCI will take care of validating regressions.


## Docker Development Environment

Project repository ships a `docker-compose.yml` file to launch an OpenLDAP and
a PostgreSQL instances.

``` console
$ docker-compose pull
...
Status: Downloaded newer image for postgres:10-alpine
$ docker-compose up -d
Creating network "ldap2pg_default" with the default driver
Creating ldap2pg_postgres_1 ...
Creating ldap2pg_ldap_1 ...
Creating ldap2pg_postgres_1
Creating ldap2pg_ldap_1 ... done
```

It's up to you to define how to access Postgres and LDAP containers from your
host: either use DNS resolution or a `docker-compose.override.yml` to expose
port on your host. Provided `docker-compose.yml` comes with
`postgres.ldap2pg.docker` and `ldap.ldap2pg.docker`
[dnsdock](https://github.com/aacebedo/dnsdock) aliases . If you want to test
SSL, you **must** access OpenLDAP through `ldap.ldap2pg.docker` domain name.

``` yaml
# contents docker-compose.override.yml
version: '3'

services:
  ldap:
    ports:
    # HOST:CONTAINER
    - 389:389
    - 636:636

  postgres:
    ports:
    - 5432:5432
```

Setup your environment with regular `PG*` envvars so that `psql` can just
connect to your PostgreSQL instance. Check with a simple `psql` call.

``` console
$ export PGHOST=postgres.ldap2pg.docker PGUSER=postgres PGPASSWORD=postgres
$ psql -c 'SELECT version()';
```

Do the same to setup `libldap2` with `LDAP*` envvars. A `ldaprc` is provided
setting up `BINDDN` and `BASE`. ldap2pg supports `LDAPPASSWORD` to set
password from env. Check it with `ldapsearch`:

``` console
$ export LDAPURI=ldaps://ldap.ldap2pg.docker LDAPPASSWORD=integral
$ ldapsearch -vxw $LDAPPASSWORD cn
# extended LDIF
#
# LDAPv3
# base <dc=ldap,dc=ldap2pg,dc=docker> (default) with scope subtree
# filter: (objectclass=*)
# requesting: cn
#

# ldap.ldap2pg.docker
dn: dc=ldap,dc=ldap2pg,dc=docker

# admin, ldap.ldap2pg.docker
dn: cn=admin,dc=ldap,dc=ldap2pg,dc=docker
cn: admin

# search result
search: 2
result: 0 Success

# numResponses: 3
# numEntries: 2
$
```

Now you can run ldap2pg from source and test your changes!

``` console
$ go run ./cmd/ldap2pg
11:10:26 INFO   Starting ldap2pg commit=(unknown) version=v5.10.0-alpha1 runtime=go1.20.3
11:10:26 WARN   ldap2pg is alpha software! Use at your own risks!
11:10:26 INFO   Using YAML configuration file. path=./ldap2pg.yml
...
11:10:27 INFO   Comparison complete. elapsed=261.3525ms mempeak=1.3MiB postgres=0s queries=459 ldap=2.33043ms searches=3
$
```

## Development Fixtures

OpenLDAP starts with `fixtures/openldap-data.ldif` loaded.
`fixtures/openldap-data.ldif` is well commented.

Some users, database and privileges are provided for testing purpose in
`fixtures/postgres.sh`. Postgres instance is initialized with this
automatically. This script also resets modifications to Postgres instance by
ldap2pg. You can run `fixtures/postgres.sh` every time you need to reset the
Postgres instance.


## Unit tests

Unit tests strictly have **no I/O**.
``` console
$ go test ./...
?       github.com/dalibo/ldap2pg/cmd/ldap2pg        [no test files]
ok      github.com/dalibo/ldap2pg/internal      (cached)
ok      github.com/dalibo/ldap2pg/internal/config       (cached)
ok      github.com/dalibo/ldap2pg/internal/inspect      (cached)
ok      github.com/dalibo/ldap2pg/internal/ldap (cached)
ok      github.com/dalibo/ldap2pg/internal/lists        (cached)
ok      github.com/dalibo/ldap2pg/internal/perf (cached)
?       github.com/dalibo/ldap2pg/internal/postgres     [no test files]
?       github.com/dalibo/ldap2pg/internal/role [no test files]
ok      github.com/dalibo/ldap2pg/internal/privilege    (cached)
ok      github.com/dalibo/ldap2pg/internal/pyfmt        (cached) [no tests to run]
ok      github.com/dalibo/ldap2pg/internal/tree (cached)
ok      github.com/dalibo/ldap2pg/internal/wanted       (cached)
$
```


## Functionnal tests

Functionnal tests tend to validate ldap2pg in real world : **no mocks**.
We put func tests in `tests/func/`.
You can run func tests right from you development environment:


``` console
$ pip install -Ur tests/func/requirements.txt
...
$ make build
$ pytest -k go --ldap2pg build/ldap2pg tests/func/
...
tests/func/test_sync.py::test_dry_run PASSED
tests/func/test_sync.py::test_real_mode PASSED
tests/func/test_sync.py::test_nothing_to_do PASSED

========================== 9 passed in 10.28 seconds ===========================
$
```

On CI, func tests are executed in CentOS 6 and 7 and RockyLinux 8.

Tests are written with the great [pytest](https://doc.pytest.org) and
[sh](https://amoffat.github.io/sh/) projects. `conftest.py` provides various
specific fixtures. The most important is that Postgres database and OpenLDAP
base are purged between each **module**. Func tests are executed in definition
order. If a test modifies Postgres, the following tests will have this
modification kept. This allows to split a big scenario in severals steps without
loosing context and CPU cycle.

Two main fixtures are very useful when testing: `psql` and `ldap`. These little
helpers provide fastpath to frequent inspection of Postgres database on LDAP
base with `sh.py`-style API.


## Documenting

[mkdocs](http://www.mkdocs.org) is in charge of building the documentation. To
edit the doc, install `docs/requirements.txt` and run `mkdocs serve` at the
toplevel directory. See [mkdocs
documentation](http://www.mkdocs.org/user-guide/writing-your-docs/) for further
information.


## Releasing

- Review `docs/changelog.md`. `# Unreleased` title will be edited.
- Increment version in `setup.py`.
- Generate release commit, tag and changelog with `make release`.
- Once CircleCI has uploaded artifacts, run `make publish-rpm` to build and publish RPM.
