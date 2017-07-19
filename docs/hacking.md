<h1>Hacking</h1>

# Development environment

A `docker-compose.yml` file is provided to launch an OpenLDAP and a PostgreSQL
instances as well as a phpLDAPAdmin to help you manage OpenLDAP.

``` console
$ docker-compose pull
...
Status: Image is up to date for dinkel/phpldapadmin:latest
$ docker-compose up -d
Creating network "ldap2pg_default" with the default driver
Creating volume "ldap2pg_ldapetc" with default driver
Creating volume "ldap2pg_ldapvar" with default driver
Creating ldap2pg_ldap_1
Creating ldap2pg_postgres_1
Creating ldap2pg_admin_1
```

It's up to you to define how to access Postgres and LDAP containers from your
host: either use DNS resolution or a `docker-compose.override.yml` to expose
port on your host. Provided `docker-compose.yml` comes with
`postgres.ldap2pg.docker` and
`ldap.ldap2pg.docker` [dnsdock](https://github.com/aacebedo/dnsdock) aliases .
If you want to test SSL, you **must** access OpenLDAP through
`ldap.ldap2pg.docker` domain name.

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
$ export LDAPURI=ldaps://ldap.ldap2pg.dockr LDAPPASSWORD=integral
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
Starting ldap2pg 2.0a1.
Using /home/bersace/src/dalibo/ldap2pg/ldap2pg.master.yml.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
Querying LDAP ou=groups,dc=ldap,dc=ldap2pg,dc=docker...
Failed to query LDAP: {'matched': 'dc=ldap,dc=ldap2pg,dc=docker', 'desc': 'No such object'}.
$
```

# Development fixtures

OpenLDAP is starts with `dev-fixture.ldif` data. `dev-fixture.ldif` is well
commented.

Some users, database and ACLs are provided for testing purpose in
`./dev-fixture.sh`. Postgres instance is initialized with this automatically.
This script also resets modifications to Postgres instance by `ldap2pg`. You can
run `./dev-fixture.sh` every time you need to reset the Postgres instance.


# Debugging

`ldap2pg` has a debug mode. Debug mode enables full logs and, if stdout is a
TTY, drops in a PDB on unhandled exception. You can enable debug mode by
exporting `DEBUG` envvar to either `1`, `y` or `Y`.

``` console
$ DEBUG=1 ldap2pg
[ldap2pg.script      DEBUG] Debug mode enabled.
[ldap2pg.config      DEBUG] Processing CLI arguments.
[ldap2pg.config       INFO] Starting ldap2pg 2.0a1.
...
[ldap2pg.script      ERROR] Unhandled error:
[ldap2pg.script      ERROR] Traceback (most recent call last):
[ldap2pg.script      ERROR]   File "/home/bersace/src/dalibo/ldap2pg/ldap2pg/script.py", line 70, in main
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

Unit tests must cover all code in `ldap2pg`.


# Functionnal tests

Functionnal tests tend to integrate `ldap2pg` in real world. No mocks. We put
func tests in `tests/func/`. You can run func tests right from you development
environment:


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

On CI, func tests are executed in a CentOS7 container, with ldap2pg and its
dependencies installed from rpm. You can reproduce this setup with
`docker-compose.yml` and some `make` calls. Run `make clean rpm tests` in
`tests/func/` to recreate rpm and test env.


``` console
$ make clean rpm tests
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

To execute tests properly, with envvars loaded, use `pytest` **make target**
inside the container.

``` console
$ make debug
docker-compose exec runner /bin/bash
[root@1dedbd5c1533 /]# cd /workspace
[root@1dedbd5c1533 workspace]# make -C tests/func/ pytest -- -x --pdb
...
(Pdb)
```

Tests are written with the great [pytest](https://doc.pytest.org)
and [sh](https://amoffat.github.io/sh/) projects. `conftest.py` provides various
specific fixtures. The most important is that Postgres database and OpenLDAP
base are purged between each module. Func tests are executed in definition
order. If a test modifies Postgres, the following tests will have this
modification kept. This allows to split a big scenario in severals steps without
loosing context and CPU cycle.

Two main fixtures are very useful when testing: `psql` and `ldap`. These little
helpers provide fastpath to recurrent inspection of Postgres database on LDAP
base with `sh.py`-style API. Also `dev` fixture resets Postgres database and
LDAP base, load the dev fixtures exposed above.

There is no code coverage in func tests, and you can't enter a debugger inside
`ldap2pg` like you do with unit tests. This is on purpose to run `ldap2pg` in
real situation. When you need to debug `ldap2pg` itself, just run it outside
pytest! **Never import `ldap2pg` in func tests**. Call it like a subprocess.


# Documenting

[mkdocs](http://www.mkdocs.org) is in charge of building the documentation. To
edit the doc, just type `mkdocs serve` at the toplevel directory and start
editing `mkdocs.yml` and `docs/`.
See [mkdocs documentation](http://www.mkdocs.org/user-guide/writing-your-docs/)
for further information.


# Packaging

We provide a recipe to build RPM package for `ldap2pg` in `packaging/`. You only
need Docker and Docker Compose.

``` console
$ make rpm
...
rpm_1  | + chown --changes --recursive 1000:1000 dist/ build/
rpm_1  | changed ownership of 'dist/ldap2pg-0.1-1.src.rpm' from root:root to 1000:1000
rpm_1  | changed ownership of 'dist/ldap2pg-0.1-1.noarch.rpm' from root:root to 1000:1000
...
$
```

You will find `.rpm` package in `dist/`. There is no repository yet, nor debian
package.
