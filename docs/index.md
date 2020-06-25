---
hide:
  - navigation
---

<h1 style="display: none"><a href="https://labs.dalibo.com/ldap2pg"><code>ldap2pg</code></a></h1>


![ldap2pg](https://github.com/dalibo/ldap2pg/raw/master/docs/img/logo-phrase.png)

Postgres is able to check password against an entreprise directory using the
LDAP protocol out of the box. ldap2pg automates the creation, update and
removal of PostgreSQL roles and users based on entreprise organigram described
in the directory.

Managing roles is close to managing privileges as you expect roles to have
proper default privileges. ldap2pg can grant and revoke privileges too.

Project goals include **stability**, **portability**, high **configurability**
and nice **user experience**.

![Screenshot](img/screenshot.png)


## Features

- Reads settings from an expressive YAML config file.
- Creates, alters and drops PostgreSQL roles from LDAP searches.
- Creates static roles from YAML to complete LDAP entries.
- Manages role members (alias *groups*).
- Grants or revokes privileges statically or from LDAP entries.
- Dry run, check mode.
- Logs LDAP searches as `ldapsearch(1)` commands.
- Logs every SQL query.


## Installation

ldap2pg requires Python 2.6+ or 3+, pyyaml, python-ldap and psycopg2.

The universal installation method is to download from PyPI using pip. Other
methods and more details are described in this documentation.

``` console
# apt install -y libldap2-dev libsasl2-dev
# pip install ldap2pg psycopg2-binary
```

ldap2pg is licensed under PostgreSQL license. ldap2pg is available with the
help of wonderful people, jump to [contributors] list to see them.

[contributors]: https://github.com/dalibo/ldap2pg/blob/master/CONTRIBUTING.md#contributors

ldap2pg **requires** a configuration file called `ldap2pg.yaml`. The [dumb but
tested
`ldap2pg.yml`](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml) is a
good way to start.

``` console
# curl -LO https://github.com/dalibo/ldap2pg/raw/master/ldap2pg.yml
# editor ldap2pg.yml
```

Finally, it's up to you to use `ldap2pg` in a crontab or a playbook. Have fun!


## Support

This documentation includes a [cookbook](cookbook) with many recipes for common
deployment pattern. If you hit a bug or didn't found what you need in
documentation, drop an [issue on
GitHub](https://github.com/dalibo/ldap2pg/issues/new)!
