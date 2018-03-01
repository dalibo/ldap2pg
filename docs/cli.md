<!--*- markdown -*-->

<h1>Command Line Interface</h1>

`ldap2pg` tries to be friendly regarding configuration. `ldap2pg` reads its
configuration from several sources, in the following order:

1. command line arguments.
2. environment variables.
3. configuration file.
4. ldaprc, ldap.conf, etc.

The `--help` switch shows regular online documentation for CLI arguments. As of
version 3.4, this looks like:

``` console
$ ldap2pg --help
usage: ldap2pg [-c PATH] [-C] [-n] [-N] [-q] [-v] [--color] [--no-color] [-?]
               [-V]

PostgreSQL roles and ACL management.

optional arguments:
  -c PATH, --config PATH
                        path to YAML configuration file (env: LDAP2PG_CONFIG).
                        Use - for stdin.
  -C, --check           check mode: exits with 1 on changes in cluster
  -n, --dry             don't touch Postgres, just print what to do (env:
                        DRY=1)
  -N, --real            real mode, apply changes to Postgres (env: DRY='')
  -q, --quiet           hide debugging messages
  -v, --verbose         add debug messages including SQL and LDAP queries
                        (env: VERBOSE)
  --color               force color output (env: COLOR=1)
  --no-color            force plain text output (env: COLOR='')
  -?, --help            show this help message and exit
  -V, --version         show version and exit

ldap2pg requires a configuration file to describe LDAP queries and role
mappings. See https://ldap2pg.readthedocs.io/en/latest/ for further details.
By default, ldap2pg runs in dry mode.
$
```

Arguments can be defined multiple times. On conflict, the last argument is used.


# Environment variables

`ldap2pg` has no CLI switch to configure Postgres connection. However, `ldap2pg`
supports `libpq` `PG*` env vars:

```
$ PGHOST=/var/run/postgresql PGUSER=postgres ldap2pg
Starting ldap2pg 2.0a2.
Using /home/src/ldap2pg/ldap2pg.yml.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
...
```

See `psql(1)` for details on `libpq` env vars. `ldap2pg` also accepts an extra
env var named `PGDSN` to define a [`libpq` connection
string](https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING):

```
$ PGDSN=postgres://postgres@localhost:5432/ ldap2pg
...
$ PGDSN="host=localhost port=5432 user=postgres" ldap2pg
...
```

`ldap2pg` works at cluster level. You **must not** specify database.


The same goes for LDAP, `ldap2pg` supports standard `LDAP*` env vars and
`ldaprc` files:

``` console
$ LDAPURI=ldaps://localhost LDAPBINDDN=cn=you,dc=entreprise,dc=fr LDAPPASSWORD=pasglop ldap2pg
Starting ldap2pg 2.0a2.
Using /home/src/ldap2pg/ldap2pg.yml.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
...
```

`ldap2pg` accepts two extra variables: `LADPPASSWORD` and `LDAPUSER`.
`LDAPPASSWORD` is self explanatory. `LDAPUSER` triggers SASL authentication.
Without `LDAPUSER`, `ldap2pg` switches to simple authentication.

See `ldap.conf(1)` for further details on how to configure.

A few other environment variables are available and described in either
[`ldap2pg.yml`](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.full.yml)
sample or CLI help.


!!! tip

    Test Postgres connexion using `psql(1)` and LDAP using `ldapsearch(1)`,
    `ldap2pg` will be okay and it will be easier to test the setup.


You can also configure Postgres and LDAP connection through `ldap2pg.yml`.
