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


## Environment variables

`ldap2pg` accepts any `PG*`
[envvars from libpq](https://www.postgresql.org/docs/current/libpq-envars.html),
all `LDAP*` envvars from libldap2. More details can be found in
[documentation](https://ldap2pg.readthedocs.io/en/latest/).

`ldap2pg` runs in `/workspace` directory in the container. Thus you can mount
configuration files in.

``` console
$ docker run --rm --volume ${PWD}:/workspace dalibo/ldap2pg:latest --verbose --dry
…
```


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
