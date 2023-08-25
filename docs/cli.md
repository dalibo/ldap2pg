<!--*- markdown -*-->

<h1>Command Line Interface</h1>

ldap2pg tries to be friendly regarding configuration and consistent with psql,
OpenLDAP utils and [12 factors apps](https://12factor.net/). ldap2pg reads its
configuration from several sources, in the following order, first prevail:

1. command line arguments.
2. environment variables.
3. configuration file.
4. ldaprc, ldap.conf, etc.

The `--help` switch shows regular online documentation for CLI arguments. As of
version 5.7, this looks like:

``` console
$ ldap2pg --help
usage: ldap2pg [OPTIONS]

      --check             Check mode: exits with 1 if Postgres instance is unsynchronized.
      --color             Force color output.
  -c, --config string     Path to YAML configuration file. Use - for stdin.
  -?, --help              Show this help message and exit. (default true)
  -q, --quiet count       Decrease log verbosity.
  -R, --real              Real mode. Apply changes to Postgres instance.
  -P, --skip-privileges   Turn off privilege synchronisation.
  -v, --verbose count     Increase log verbosity.
  -V, --version           Show version and exit. (default true)


By default, ldap2pg runs in dry mode.
ldap2pg requires a configuration file to describe LDAP searches and mappings.
See https://ldap2pg.readthedocs.io/en/latest/ for further details.
```

Arguments can be defined multiple times. On conflict, the last argument is used.


## Environment variables

ldap2pg has no CLI switch to configure Postgres connection.
However, ldap2pg supports `libpq` [PG* env vars](https://www.postgresql.org/docs/current/libpq-envars.html).

See [psql(1)] for details on libpq env vars.

[psql(1)]: https://www.postgresql.org/docs/current/app-psql.html#APP-PSQL-ENVIRONMENT

The same goes for LDAP, ldap2pg supports standard `LDAP*` env vars and `ldaprc` files.
See `ldap.conf(5)` for further details on how to configure.
ldap2pg accepts one extra variable: `LDAPPASSWORD`.

!!! tip

    Test Postgres connexion using `psql(1)` and LDAP using `ldapwhoami(1)`,
    ldap2pg will be okay
    and it will be easier to debug the setup and the   configuration later.


## Logging setup

ldap2pg have several levels of logging:

- `ERROR`: error details. When this happend, ldap2pg will crash.
- `WARNING`: ldap2pg warns about choices you should be aware of.
- `CHANGE`: only changes applied to Postgres cluster. (aka Magnus Hagander level).
- `INFO` (default): tells what ldap2pg is doing, especially before long task.
- `DEBUG`: everything, including raw SQL queries and LDAP searches and
  introspection details.

The `--quiet` and `--verbose` switches respectively decrease and increase
verbosity.

You can select the highest level of verbosity with `LDAP2PG_VERBOSITY` envvar. For example:


``` console
$ LDAP2PG_VERBOSITY=DEBUG ldap2pg
12:23:45 INFO   Starting ldap2pg                                 version=v6.0-alpha5 runtime=go1.21.0 commit=<none>
12:23:45 WARN   Running a prerelease! Use at your own risks!
12:23:45 DEBUG  Searching configuration file in standard locations.
12:23:45 DEBUG  Found configuration file.                        path=./ldap2pg.yml
$
```

ldap2pg output varies whether it's running with a TTY or not.
If standard error is a TTY, logging is colored and tweaked for human reading.
Otherwise, logging format is pure logfmt, for machine processing.
You can force human-readable output by using `--color` CLI switch.
