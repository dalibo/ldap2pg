<p align="center">
  <a href="https://labs.dalibo.com/ldap2pg" rel="nofollow" class="rich-diff-level-one">
    <img alt="ldap2pg: PostgreSQL role and privileges management" src="https://github.com/dalibo/ldap2pg/raw/master/docs/img/logo-phrase.png"/>
  </a>
</p>

<p align="center">
  <strong>Swiss-army knife to synchronize Postgres roles and privileges from YAML or LDAP.</strong>
</p>

<p align="center">
  <a href="https://ldap2pg.rtfd.io/" rel="nofollow" class="rich-diff-level-one">
    <img src="https://readthedocs.org/projects/ldap2pg/badge/?version=latest" alt="Documentation" />
  </a>
  <a href="https://circleci.com/gh/dalibo/ldap2pg" rel="nofollow" class="rich-diff-level-one">
    <img src="https://circleci.com/gh/dalibo/ldap2pg.svg?style=shield" alt="Continuous Integration report" />
  </a>
  <a href="https://hub.docker.com/r/dalibo/ldap2pg" rel="nofollow" class="rich-diff-level-one">
    <img alt="Docker Cloud Automated build" src="https://img.shields.io/docker/cloud/automated/dalibo/ldap2pg">
  </a>
</p>


Postgres is able to check password of an existing role using the LDAP protocol out of the box.
ldap2pg automates the creation, update and removal of PostgreSQL roles and users from an entreprise directory.

Managing roles is close to managing privileges as you expect roles to have proper default privileges.
ldap2pg can grant and revoke privileges too.


# Features

- Reads settings from an expressive YAML config file.
- Creates, alters and drops PostgreSQL roles from LDAP searches.
- Creates static roles from YAML to complete LDAP entries.
- Manages role parents (alias *groups*).
- Grants or revokes privileges statically or from LDAP entries.
- Dry run, check mode.
- Logs LDAP searches as `ldapsearch(1)` commands.
- Logs **every** SQL statements.

Here is a sample configuration and execution:

``` console
$ cat ldap2pg.yml
version: 6

rules:
- role:
    name: nominal
    options: NOLOGIN
    comment: "Database owner"
- ldapsearch:
    base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
    filter: "(objectClass=organizationalPerson)"
  role:
    name: '{cn}'
    options:
      LOGIN: yes
      CONNECTION LIMIT: 5
$ ldap2pg --real
08:25:12 INFO   Starting ldap2pg                                 version=v6.0-alpha5 runtime=go1.21.0 commit=<none>
08:25:12 INFO   Using YAML configuration file.                   path=docs/readme/ldap2pg.yml
08:25:12 INFO   Running as unprivileged user.                    user=ldap2pg super=false server="PostgreSQL 15.3" cluster=ldap2pg-dev database=nominal
08:25:12 INFO   Connected to LDAP directory.                     uri=ldaps://ad.bridoulou.fr authzid="dn:cn=administrator,cn=users,dc=bridoulou,dc=fr"
08:25:12 INFO   Real mode. Postgres instance will modified.
08:25:12 CHANGE Create role.                                     role=charles database=nominal
08:25:12 CHANGE Set role comment.                                role=charles database=nominal
08:25:12 CHANGE Inherit role for management.                     role=charles database=nominal
08:25:12 CHANGE Alter options.                                   role=alain options="LOGIN CONNECTION LIMIT 5" database=nominal
08:25:12 CHANGE Terminate running sessions.                      role=omar database=nominal
08:25:12 CHANGE Allow current user to reassign objects.          role=omar parent=ldap2pg database=nominal
08:25:12 CHANGE Reassign objects and purge ACL.                  role=omar owner=nominal database=nominal
08:25:12 CHANGE Drop role.                                       role=omar database=nominal
08:25:12 INFO   Comparison complete.                             elapsed=68.47058ms mempeak=1.6MiB postgres=15.323294ms queries=8 ldap=635.894µs searches=1
$
```


# Installation

Download package or binary from [Releases page](https://github.com/dalibo/ldap2pg/releases).

``` console
$ sudo yum install https://github.com/dalibo/ldap2pg/releases/download/v6.1/ldap2pg_6.1_linux_amd64.rpm
...
Installed:
  ldap2pg-6.1-1.x86_64

Complete!
$ ldap2pg --help
usage: ldap2pg [OPTIONS] [dbname]

      --check              Check mode: exits with 1 if Postgres instance is unsynchronized.
      --color              Force color output. (default true)
  -c, --config string      Path to YAML configuration file. Use - for stdin.
  -C, --directory string   Path to directory containing configuration files.
  -?, --help               Show this help message and exit.
  -q, --quiet count        Decrease log verbosity.
  -R, --real               Real mode. Apply changes to Postgres instance.
  -P, --skip-privileges    Turn off privilege synchronisation.
  -v, --verbose count      Increase log verbosity.
  -V, --version            Show version and exit.


Optional argument dbname is alternatively the database name or a conninfo string or an URI.
See man psql(1) for more information.

By default, ldap2pg runs in dry mode.
ldap2pg requires a configuration file to describe LDAP searches and mappings.
See https://ldap2pg.readthedocs.io/en/latest/ for further details.
$
```

`ldap2pg` is licensed under [PostgreSQL license](https://opensource.org/licenses/postgresql).

ldap2pg **requires** a configuration file called `ldap2pg.yaml`.
Project ships a [tested ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml) as a starting point.

``` console
# curl -LO https://github.com/dalibo/ldap2pg/raw/master/ldap2pg.yml
# editor ldap2pg.yml
```

Finally, it's up to you to use `ldap2pg` in a crontab or a playbook.
Have fun!

`ldap2pg` is reported to work with [Samba Directory](https://www.samba.org/),
[OpenLDAP](https://www.openldap.org/),
[FreeIPA](https://www.freeipa.org/),
Oracle Internet Directory and
Microsoft Active Directory.


# Support

If you need support
and you didn't found it in [documentation](https://ldap2pg.readthedocs.io/),
just drop a question in a [GitHub discussion](https://github.com/dalibo/ldap2pg/discussions)!
French accepted.
If you hit a bug or miss a feature, open a [GitHub issue](https://github.com/dalibo/ldap2pg/issues/new).
Don't miss the [cookbook](https://ldap2pg.readthedocs.io/en/latest/cookbook/) for advanced use cases.


# Authors

ldap2pg is a [Dalibo Labs](https://labs.dalibo.com/) project.

- [Étienne BERSAC](https://github.com/bersace) is the maintainer.
- [Damien Cazeils](https://www.damiencazeils.com) designed the logo.
- [Harold le CLÉMENT de SAINT-MARCQ](https://github.com/hlecleme) implemented LDAP sub searches.
- [Randolph Voorhies](https://github.com/randvoorhies) implemented role configuration synchronization.
