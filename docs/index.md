<h1><code>ldap2pg</code></h1>

`ldap2pg` is a simple yet powerful tool to manage Postgres roles and privileges
statically or from LDAP directories, including OpenLDAP and Active Directory.

Project goals include **stability**, **portability**, high **configurability**,
state of the art code **quality** and nice **user experience**.

![Screenshot](img/screenshot.png)


## Highlighted features

- Creates, alter and drops PostgreSQL roles from LDAP queries.
- Creates static roles from YAML to complete LDAP entries.
- Manage role members (alias *groups*).
- Grant or revoke privileges statically or from LDAP entries.
- Dry run.
- Logs LDAP queries as `ldapsearch` commands.
- Logs **every** SQL queries.
- Reads settings from an expressive YAML config file.


## Quick installation

Just use PyPI as any regular Python project:

``` console
# apt install -y libldap2-dev libsasl2-dev
# pip3 install ldap2pg
# ldap2pg --help
```

Now you **must** configure [Postgres](cookbook.md#configure-postgres-connection)
and [LDAP](cookbook.md#query-ldap) connections, then synchronisation map in
[`ldap2pg.yml`](config.md). The [dumb but tested
`ldap2pg.yml`](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml) is a
good way to start.

``` console
# curl -LO https://github.com/dalibo/ldap2pg/raw/master/ldap2pg.yml
# editor ldap2pg.yml
```

Finally, it's up to you to use `ldap2pg` in a crontab or a playbook. Have fun!
