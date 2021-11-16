[![ldap2pg: PostgreSQL role and privileges management](https://github.com/dalibo/ldap2pg/raw/master/docs/img/logo-phrase.png)](https://labs.dalibo.com/ldap2pg)

Swiss-army knife to synchronize Postgres roles and privileges from YAML
or LDAP.


## Features

- Creates, alters and drops PostgreSQL roles from LDAP queries.
- Creates static roles from YAML to complete LDAP entries.
- Manages role members (alias *groups*).
- Grants or revokes privileges statically or from LDAP entries.
- Dry run.
- Logs LDAP queries as `ldapsearch` commands.
- Logs **every** SQL query.
- Reads settings from an expressive YAML config file.


## How to use this image

`ldap2pg` runs in `/workspace` directory in the container. Thus you can mount
configuration files in.

``` console
$ docker run --rm --volume ${PWD}:/workspace dalibo/ldap2pg:latest --verbose --dry
```

Or use `LDAP2PG_CONFIG` environment variable or `--config` CLI switch to point
to another path.


## Environment variables

`ldap2pg` accepts any `PG*`
[envvars from libpq](https://www.postgresql.org/docs/current/libpq-envars.html),
all `LDAP*` envvars from libldap2. More details can be found in
[documentation](https://ldap2pg.readthedocs.io/en/latest/).


## Docker Secrets

As an alternative to passing sensitive information via environment variables,
`_FILE` may be appended to some environment variables, causing the
initialization script to load the values for those variables from files present
in the container. In particular, this can be used to load passwords from Docker
secrets stored in `/run/secrets/<secret_name>` files. For example:

``` console
$ docker run --rm -e PGPASSWORD_FILE=/run/secrets/postgres-passwd dalibo/ldap2pg:latest --verbose --dry
```

Currently, this is only supported for `PGPASSWORD`and `LDAPPASSWORD`.


## Initialization Scripts

If you would like to do additional initialization in an image derived from this
one, add one or more `*.sh` scripts under /docker-entrypoint.d (creating the
directory if necessary). Before the entrypoint calls ldap2pg, it will run any
executable *.sh scripts, and source any non-executable *.sh scripts found in
that directory.

These initialization files will be executed in sorted name order as defined by
the current locale, which defaults to en_US.utf8.


## Support

If you need support and you didn\'t found it in
[documentation](https://ldap2pg.readthedocs.io/en/latest/), just drop a question
in a [GitHub issue](https://github.com/dalibo/ldap2pg/issues/new)! Don\'t miss
the [cookbook](https://ldap2pg.readthedocs.io/en/latest/cookbook/). You\'re
welcome!

`ldap2pg` is licensed under
[PostgreSQL license](https://opensource.org/licenses/postgresql). `ldap2pg` is
available with the help of wonderful people, jump to
[contributors](https://github.com/dalibo/ldap2pg/blob/master/CONTRIBUTING.md#contributors)
list to see them.
