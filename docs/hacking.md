<h1>Hacking</h1>

You are welcome to contribute to `ldap2pg` with patch to code, documentation or
configuration sample ! Here is an extended documentation on how to setup *a*
development environment. Feel free to adapt to your cumfort. Automatic tests on
CircleCI will take care of validating regressions.


# Docker Development Environment

A `docker-compose.yml` file is provided to launch an OpenLDAP and a PostgreSQL
instances as well as a phpLDAPAdmin to help you manage OpenLDAP.

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
setting up `BINDDN` and `BASE`. `ldap2pg` supports `LDAPPASSWORD` to set
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

Now you can install `ldap2pg` from source and test your changes!

``` console
$ pip install -e .
$ ldap2pg
Starting ldap2pg 3.4.
Using /home/bersace/src/dalibo/ldap2pg/ldap2pg.yml.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
Querying LDAP cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker...
Querying LDAP ou=groups,dc=ldap,dc=ldap2pg,dc=docker...
Would create albert.
...
Comparison complete.
$
```

# Development Fixtures

OpenLDAP starts with `fixture/openldap-data.ldif` loaded.
`fixture/openldap-data.ldif` is well commented.

Some users, database and privileges are provided for testing purpose in
`./fixtures/postgres.sh`. Postgres instance is initialized with this
automatically. This script also resets modifications to Postgres instance by
`ldap2pg`. You can run `fixtures/postgres.sh` every time you need to reset the
Postgres instance.


# Debugging

`ldap2pg` has a debug mode. Debug mode enables full logs and, if stdout is a
TTY, drops in a PDB on unhandled exception. You can enable debug mode by
exporting `DEBUG` envvar to either `1`, `y` or `Y`.

``` console
$ DEBUG=1 ldap2pg
$ DEBUG=1 ldap2pg
[ldap2pg.script      DEBUG] Debug mode enabled.
[ldap2pg.config      DEBUG] Processing CLI arguments.
[ldap2pg.config       INFO] Starting ldap2pg 3.4.
[ldap2pg.config      DEBUG] Trying ./ldap2pg.yml.
[ldap2pg.config       INFO] Using /home/bersace/src/dalibo/ldap2pg/ldap2pg.yml.
[ldap2pg.config      DEBUG] Read verbose from DEBUG.
[ldap2pg.config      DEBUG] Read ldap:uri from LDAPURI.
[ldap2pg.config      DEBUG] Read ldap:password from LDAPPASSWORD.
[ldap2pg.config      DEBUG] Read postgres:dsn from PGDSN.
[ldap2pg.config      DEBUG] Read sync_map from YAML.
...
[ldap2pg.script      ERROR] Unhandled error:
[ldap2pg.script      ERROR] Traceback (most recent call last):
[ldap2pg.script      ERROR]   File ".../ldap2pg/script.py", line 70, in main
[ldap2pg.script      ERROR]     wrapped_main(config)
...
[ldap2pg.script      ERROR]     raise ValueError(...)
[ldap2pg.script      ERROR] ValueError: ...
[ldap2pg.script      DEBUG] Dropping in debugger.
> /home/../.local/share/virtualenvs/l2p/lib/python3.5/site-packages/...
-> raise ValueError(...)
(Pdb) _
```


# Unit tests

Unit tests strictly have **no I/O**. We use pytest to execute them. Since we
also have a functionnal test battery orchestrated with pytest, you must scope
pytest execution to `tests/unit/`.

``` console
$ pip install -Ur requirements-ci.txt
...
$ pytest tests/unit/
============================= test session starts ==============================
...
ldap2pg/psql.py          71      0   100%
ldap2pg/role.py          96      0   100%
ldap2pg/script.py        59      0   100%
ldap2pg/utils.py         26      0   100%
---------------------------------------------------
TOTAL                   870      0   100%


========================== 72 passed in 0.49 seconds ===========================
$
```

Unit tests must cover all code in `ldap2pg`. We use
[CodeCov](https://codecov.io/) to enforce this.


# Functionnal tests

Functionnal tests tend to validate `ldap2pg` in real world : **no mocks**. We
put func tests in `tests/func/`. You can run func tests right from you
development environment:


``` console
$ pip install -Ur requirements-ci.txt
...
$ pytest tests/func/
...
tests/func/test_sync.py::test_dry_run PASSED
tests/func/test_sync.py::test_real_mode PASSED
tests/func/test_sync.py::test_nothing_to_do PASSED

========================== 9 passed in 10.28 seconds ===========================
$
```

On CI, func tests are executed in CentOS 6 and CentOS 7, with ldap2pg and its
dependencies installed from rpm. You can reproduce this setup with
`docker-compose.yml` and some `make` calls. Run `make -C tests/func/ clean rpm
tests` to recreate rpm and test env.


``` console
$ make -C tests/func/ clean rpm tests
runner_1    |
runner_1    | ========================== 9 passed in 18.16 seconds ===========================
runner_1    | make: Leaving directory `/workspace/tests/func'
runner_1    | + teardown
runner_1    | + '[' -z '' -a 0 -gt 0 -a 1 = 1 ']'
func_runner_1 exited with code 0
Aborting on container exit...
$
```

On failure, the container waits forever like this:

``` console
$ make tests
...
runner_1    | ===================== 1 failed, 8 passed in 23.47 seconds ======================
runner_1    | make: *** [pytest] Error 1
runner_1    | make: Leaving directory `/workspace/tests/func'
runner_1    | + teardown
runner_1    | + '[' -z '' -a 2 -gt 0 -a 1 = 1 ']'
runner_1    | + tailf /dev/null
```

This way you can either kill it with `^C` or enter it to debug. Run `make debug`
to enter the container and start debugging it. Source tree is mounted at
`/workspace`. To reduce dev loop, just `pip install -e .` to use WIP code rather
than rpm version.

``` console
$ make debug
docker-compose exec runner /bin/bash
[root@1dedbd5c1533 /]# cd /workspace
[root@1dedbd5c1533 workspace]# pytest -x tests/func/ --pdb
...
(Pdb)
```

Tests are written with the great [pytest](https://doc.pytest.org) and
[sh](https://amoffat.github.io/sh/) projects. `conftest.py` provides various
specific fixtures. The most important is that Postgres database and OpenLDAP
base are purged between each **module**. Func tests are executed in definition
order. If a test modifies Postgres, the following tests will have this
modification kept. This allows to split a big scenario in severals steps without
loosing context and CPU cycle.

Two main fixtures are very useful when testing: `psql` and `ldap`. These little
helpers provide fastpath to frequent inspection of Postgres database on LDAP
base with `sh.py`-style API. Also `dev` fixture resets Postgres database and
LDAP base and loads the dev fixtures exposed above.

There is no code coverage in func tests, and you can't enter a debugger inside
`ldap2pg` like you do with unit tests. This is on purpose to run `ldap2pg` in
real situation. When you need to debug `ldap2pg` itself, just run it outside
pytest! **Never import `ldap2pg` in func tests**. Call it like a subprocess.
Logs should be enough to diagnose errors.


# Documenting

[mkdocs](http://www.mkdocs.org) is in charge of building the documentation. To
edit the doc, just type `mkdocs serve` at the toplevel directory and start
editing `mkdocs.yml` and `docs/`. See [mkdocs
documentation](http://www.mkdocs.org/user-guide/writing-your-docs/) for further
information.


# Releasing

- Review `docs/changelog.md`. `# Unreleased` title will be edited.
- Increment version in `setup.py`.
- Generate release commit, tag and changelog with `make release`.
- Upload source tarball and RPM with `make upload`.
