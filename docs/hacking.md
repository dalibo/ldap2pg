---
hide:
  - navigation
---

<h1>Hacking</h1>

You are welcome to contribute to ldap2pg with patch to code, documentation or configuration sample !
Here is an extended documentation on how to setup *a* development environment.
Feel free to adapt to your cumfort.
Automatic tests on CircleCI will take care of validating regressions.


## Docker Development Environment

Project repository ships a `docker-compose.yml` file to launch an OpenLDAP and a PostgreSQL instances.

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

It's up to you to define how to access Postgres and LDAP containers from your host:
either use DNS resolution or a `docker-compose.override.yml` to expose port on your host.
Provided `docker-compose.yml` comes with `postgres.ldap2pg.docker` and `ldap.ldap2pg.docker` [dnsdock](https://github.com/aacebedo/dnsdock) aliases.
If you want to test SSL, you **must** access OpenLDAP through `ldap.ldap2pg.docker` domain name.

Setup your environment with regular `PG*` envvars so that `psql` can just connect to your PostgreSQL instance.
Check with a simple `psql` invocation.

``` console
$ export PGHOST=postgres.ldap2pg.docker PGUSER=postgres PGPASSWORD=postgres
$ psql -c 'SELECT version()';
```

Do the same to setup `libldap2` with `LDAP*` envvars.
A `ldaprc` is provided setting up `BINDDN` and `BASE`.
ldap2pg supports `LDAPPASSWORD` to set password from env.
Check it with `ldapsearch`:

``` console
$ export LDAPURI=ldaps://samba1.ldap2pg.docker LDAPPASSWORD=integral
$ ldapsearch -vxw $LDAPPASSWORD -s base cn
ldap_initialize( <DEFAULT> )
filter: (objectclass=*)
requesting: cn
# extended LDIF
#
# LDAPv3
# base <cn=users,dc=bridoulou,dc=fr> (default) with scope baseObject
# filter: (objectclass=*)
# requesting: cn
#

# Users, bridoulou.fr
dn: CN=Users,DC=bridoulou,DC=fr
cn: Users

# search result
search: 2
result: 0 Success

# numResponses: 2
# numEntries: 1
$
```

### Environement without DNS resolution

To access OpenLDAP and PostgreSQL without dnsdock,
exposes containers ports to your host with the following override:

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

Use `PGHOST=localhost` and `LDAPURI=ldap://localhost:389`.


### Running ldap2pg with Changes

Now you can run ldap2pg from source and test your changes!

``` console
$ go run ./cmd/ldap2pg
09:54:27 INFO   Starting ldap2pg                                 version=v6.0-alpha5 runtime=go1.20.3 commit=<none>
09:54:27 WARN   Running a prerelease! Use at your own risks!
09:54:27 INFO   Using YAML configuration file.                   path=./ldap2pg.yml
...
09:54:27 INFO   Nothing to do.                                   elapsed=78.470278ms mempeak=1.2MiB postgres=0s queries=0 ldap=486.921µs searches=1
$
```

## Development Fixtures

ldap2pg project comes with three cases for testing:

  - nominal: a regular case with:
    - running unprivileged
    - a single database named `nominal`.
    - 3 groupes : readers, writers and owners
    - roles and privileges synchronized.
  - extra: few corner cases together
    - running as superuser
    - synchronize role configuration
    - do LDAP sub-searches.
  - big: a huge synchronization project
    - multiple databases with a LOT of schemas, tables, views, etc.
    - all privileges synchronized
    - 3 groups per schemas.
    - 1K users in directory.

`test/fixtures/` holds fixtures for OpenLDAP et PostgreSQL.
Default development environment loads nominal and extra fixtures.
By default, big case is not loaded.
Func tests use nominal and extra fixtures.
See below for big case.

`test/fixtures/reset.sh` resets PostgreSQL state.
You can also use `make reset-postgres` to recreate PostgreSQL container from scratch.


## Unit tests

Unit tests strictly have **no I/O**.
Run unit tests as usual go tests.

``` console
$ go test ./...
?       github.com/dalibo/ldap2pg/cmd/ldap2pg   [no test files]
?       github.com/dalibo/ldap2pg/cmd/render-doc        [no test files]
ok      github.com/dalibo/ldap2pg/cmd/mon-dojo  0.002s
ok      github.com/dalibo/ldap2pg/internal      0.003s
ok      github.com/dalibo/ldap2pg/internal/config       0.007s
ok      github.com/dalibo/ldap2pg/internal/inspect      0.005s
ok      github.com/dalibo/ldap2pg/internal/ldap 0.005s
ok      github.com/dalibo/ldap2pg/internal/lists        0.005s
?       github.com/dalibo/ldap2pg/internal/postgres     [no test files]
ok      github.com/dalibo/ldap2pg/internal/perf 0.004s
ok      github.com/dalibo/ldap2pg/internal/privilege    0.003s
?       github.com/dalibo/ldap2pg/internal/role [no test files]
ok      github.com/dalibo/ldap2pg/internal/pyfmt        0.004s
ok      github.com/dalibo/ldap2pg/internal/tree 0.002s
ok      github.com/dalibo/ldap2pg/internal/wanted       0.003s
$
```


## Functionnal tests

`test/` directory is a [pytest] project with functionnal tests.
Functionnal tests tend to validate ldap2pg in real world : **no mocks**.

Func tests requires Python 3.6.
Create a virtualenv to isolate ldap2pg dev Python dependencies.
Install dev dependencies with `pip install -Ur test/requirements.txt`.


``` console
$ pip install -Ur test/requirements.txt
...
Successfully installed iniconfig-2.0.0 packaging-23.1 pluggy-1.3.0 pytest-7.4.2 sh-1.14.1
$
```

You can run func tests right from you development environment:


``` console
$ pip install -Ur test/requirements.txt
...
$ pytest test/
...
ldap2pg: /home/bersace/src/dalibo/ldap2pg/test/ldap2pg.sh
...
test/test_nominal.py::test_re_revoke PASSED                                  [ 90%]
test/test_nominal.py::test_nothing_to_do PASSED                              [100%]

=============================== 11 passed in 14.90s ================================
$
```

CI executes func tests in CentOS 6 and 7 and RockyLinux 8 and 9.

Tests are written with the great [pytest](https://doc.pytest.org) and [sh](https://amoffat.github.io/sh/) projects.
`conftest.py` provides various specific fixtures.
The most important is that Postgres database and OpenLDAP base are purged between each **module**.
pytests executes Func tests in definition order.
If a test modifies Postgres, the following tests will have this modification kept until the end of the module.
This allows to split a big scenario in severals steps without loosing context and CPU cycle.

Two main pytest fixtures are very useful when testing: `psql` and `ldap`.
These little helpers provide fastpath to frequent inspection of Postgres database on LDAP base with `sh.py`-style API.


## Big Case

To stress ldap2pg on big setup, use `make big`.
This will feed directory with a lot of users and groups, several databases with a lot of schemas, etc.
Synchronize this setup with:

``` console
$ test/genperfconfig.sh | PGDATABASE=big0 go run ./cmd/ldap2pg -c -
```


## Documenting

Building documentation requires Python 3.7.
[mkdocs](http://www.mkdocs.org) is in charge of building the documentation.
To edit the doc, install `docs/requirements.txt` and run `mkdocs serve` at the toplevel directory.
See [mkdocs documentation](http://www.mkdocs.org/user-guide/writing-your-docs/) for further information.

``` console
$ pip install -r docs/requirements.txt
...
Successfully installed babel-2.12.1 certifi-2023.7.22 charset-normalizer-3.2.0 click-8.1.7 colorama-0.4.6 ghp-import-2.1.0 idna-3.4 jinja2-3.1.2 markdown-3.5 m...
```


## Releasing

- Review `docs/changelog.md`.
  `# Unreleased` title will be edited.
- Increment version in `internal/VERSION`.
- Generate release commit, tag and changelog with `make release`.
- Once CircleCI has created GitHub release artifacts, publish packages with `make publish-packages`.
- Once Docker Hub has published new tag, tag latest image on docker hub with `make tag-latest`.
- Increment `internal/VERSION` to a development version.
  Commit and push to master.
