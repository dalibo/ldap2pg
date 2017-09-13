<h1>Configuration</h1>

`ldap2pg` tries to be friendly regarding configuration. `ldap2pg` reads its
configuration from several sources, in the following order:

1. command line arguments.
2. environment variables.
3. configuration file.
4. ldaprc, ldap.conf, etc.

The `--help` switch shows regular online documentation for CLI arguments. As of
version 2.0, this looks like:

``` console
$ ldap2pg --help
usage: ldap2pg [-c PATH] [-n] [-N] [-v] [--color] [--no-color] [-?] [-V]

PostgreSQL roles and ACL management.

optional arguments:
  -c PATH, --config PATH
                        path to YAML configuration file (env: LDAP2PG_CONFIG).
                        Use - for stdin.
  -n, --dry             don't touch Postgres, just print what to do (env:
                        DRY=1)
  -N, --real            real mode, apply changes to Postgres (env: DRY='')
  -v, --verbose         add debug messages including SQL and LDAP queries
                        (env: VERBOSE)
  --color               force color output (env: COLOR=1)
  --no-color            force plain text output (env: COLOR='')
  -?, --help            show this help message and exit
  -V, --version         show version and exit

ldap2pg requires a configuration file to describe LDAP queries and role
mappings. See https://ldap2pg.readthedocs.org/ for further details. By
default, ldap2pg runs in dry mode.
```

Arguments can be defined multiple times. On conflict, the last argument is used.


# Environment variables

`ldap2pg` has no CLI switch to configure LDAP and Postgres connexion. However,
`ldap2pg` supports `libpq` `PG*` env vars:

```
$ PGHOST=/var/run/postgresql PGUSER=postgres ldap2pg
Starting ldap2pg 2.0a2.
Using /home/src/ldap2pg/ldap2pg.yml.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
...
```

See `psql(1)` for details on `libpq` env vars. `ldap2pg` also accepts an extra
env var named `PGDSN` to define
a
[`libpq` connection string](https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING):

```
$ PGDSN=postgres://postgres@localhost:5432/ ldap2pg
...
$ PGDSN="host=localhost port=5432 user=postgres" ldap2pg
...
```

`ldap2pg` works at cluster level. You **must not** specify database.


The same goes for LDAP, `ldap2pg` supports `LDAP*` env vars and `ldaprc` file:

``` console
$ LDAPURI=ldaps://localhost LDAPBINDDN=cn=you,dc=entreprise,dc=fr LDAPPASSWORD=pasglop ldap2pg
Starting ldap2pg 2.0a2.
Using /home/src/ldap2pg/ldap2pg.yml.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
...
```

`ldap2pg` accepts two extras variables: `LADPPASSWORD` and `LDAPUSER`.
`LDAPPASSWORD` is self explanatory. `LDAPUSER` triggers SASL authentication.
Without `LDAPUSER`, `ldap2pg` switches to simple authentication.

See `ldap.conf(1)` for further details on how to configure.

A few other environment variables are available and described in
either
[`ldap2pg.yml`](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml)
sample or CLI help.


You can also configure Postgres and LDAP connection through `ldap2pg.yml`.


# `ldap2pg.yml`

`ldap2pg` **requires** a config file where the synchronization map is described.
Everything can be configured from the YAML file: verbosity, real mode, LDAP and
Postgres credentials, ACL and synchronization map.


## File location

`ldap2pg` searches for files in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set, the standard file locations are
ignored. You can specify `-` to read configuration from standard input. This is
helpful to feed `ldap2pg` with dynamic configuration.

``` console
$ ls *.yml
ldap2pg.yml
$ ldap2pg --config my-inexistant-file.yml
Cannot access configuration file my-inexistant-file.yml.
$
```

## YAML format

The first section contains various parameters for `ldap2pg` behaviour.

``` yaml
# Colorization. env var: COLOR=<anything>
color: yes

# Verbose messages. Includes SQL and LDAP queries. env var: VERBOSE
verbose: no

# Dry mode. env var: DRY=<anything>
dry: yes
```

The next two sections define connexion parameters for LDAP and Postgres. Beware
that `ldap2pg` refuses to read a password from a group readable or world
readable `ldap2pg.yml`.

``` yaml
ldap:
  uri: ldap://ldap2pg.local:389
  binddn: cn=admin,dc=ldap2pg,dc=local
  user: saslusername
  password: SECRET

postgres:
  dsn: postgres://user@%2Fvar%2Frun%2Fpostgresql:port/
```


## Defining ACL

The key `acl_dict` references all known ACL definitions. An ACL is loosely
defined in `ldap2pg`. It's actually just a name associated with three queries:
`inspect`, `grant` and `revoke`.

`inspect` query is called **once** per database in the cluster. It must return a
rowset with three columns: the first is the schema name, the second is the role
and the last is a boolean indicating whether **every** ACL covered by the GRANT
are granted. Schema name can be `NULL` if the schema is irrelevant. Each tuple
in the rowset references a grant of this ACL to a role on a schema (or none).

`inspect` can be undefined. This is just as if the query returns an empty
rowset. It's actually a bad idea no to provide `inspect`. This won't allow
`ldap2pg` to revoke ACL. Also, this prevents you to check that a cluster is
synchronized: `ldap2pg` will always re-grant the ACL.

`grant` and `revoke` provide queries to respectively grant and revoke the ACL.
The query is formatted with three parameters: `database`, `schema` and `role`.
`database` strictly equals to `CURRENT_DATABASE`, it's just there to help
putting identifier in the query. `ldap2pg` uses Python's [*Format String
Syntax*](https://docs.python.org/3.7/library/string.html#formatstrings).
See example below. In verbose mode, you will see the formatted queries.

Here is an example of a simple ACL which is not schema-aware:

``` yaml
acl_dict:
  connect:
    inspect: |
      WITH d AS (
          SELECT
              (aclexplode(datacl)).grantee AS grantee,
              (aclexplode(datacl)).privilege_type AS priv
          FROM pg_catalog.pg_database
          WHERE datname = current_database()
      )
      SELECT NULL as namespace, r.rolname, TRUE AS complete
      FROM pg_catalog.pg_roles AS r
      JOIN d ON d.grantee = r.oid AND d.priv = 'CONNECT'
    grant: |
      GRANT CONNECT ON DATABASE {database} TO {role};
    revoke: |
      REVOKE CONNECT ON DATABASE {database} FROM {role}
```

Writing `inspect` queries requires a deep knowledge of Postgres internals. See
[System Catalogs](https://www.postgresql.org/docs/current/static/catalogs.html)
section in PostgreSQL documentation to see how ACL are actually stored in
Postgres. Checking whether a `GRANT SELECT ON ALL TABLES IN SCHEMA` is complete
is rather tricky. See [Cookbook](cookbook.md) for detailed and real use case.


## Grouping ACL

Privileges are often granted together. E.g. you grant `SELECT ON ALL TABLES IN
SCHEMA` along `ALTER DEFAULT PRIVILEGES IN SCHEMA SELECT ON ALL TABLES`. This
can be very tricky to aggregate all ACL inspection in a single query. To help in
this situation, `ldap2pg` manage *groups* of ACL, defined in `acl_groups` entry.

```
acl_dict:
  select: {...}
  default-select: {...}

acl_groups:
  ro: [select, default-select]
```

Now you can use `ro` as a regular ACL in synchronization map. See the
[Cookbook](cookbook.md) for examples.


## Synchronization map

The synchronization map is a complex and polyform setting. The core element of
the synchronization map is called a *mapping*. One or more mappings can be
associated with either the cluster, a database within the cluster or a single
schema in one database.

A mapping is a dict with three kind of rules: `ldap`, `roles` and `grant`.
`ldap` entry is optionnal, however either one of `roles` or `grant` is required.
Here is a sample:

``` yaml 
- ldap:
    base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
    filter: "(objectClass=organizationalRole)"
    scope: sub
    attribute: cn
  role:
    name_attribute: cn
    options: LOGIN
  grant:
    role_attribute: cn
    acl: ro
```


`ldap` define the LDAP query. `base`, `filter` and either `attribute` or
`attributes` must be defined. `scope` defaults to `sub`. Their meaning is the
same as in `ldapsearch`. `attribute` can be either a string or a list of
strings. You need at least one attribute, an empty entry is useless.

`role` or `roles` contains one or more rules to
generate
[Postgres role](https://www.postgresql.org/docs/current/static/user-manag.html)
for each entry returned by the LDAP search.

`grant` contains one or more rules to tell `ldap2pg` who to grant an ACL to. See
below for details.

An LDAP search is not mandatory. `ldap2pg` can create roles defined statically
from YAML. Ensure role and grant rules in the mapping do not have `*_attribute`
keys. They will try to refer to an inexisting LDAP entry.

Each LDAP search is done once and only once. There is no loop neither
deduplication of LDAP searches.


### `role` parameters

`name` or `names` contains one or more static name of role you want to define in
Postgres. This is usefull to e.g. define a `ldap_users` group. `names` parameter
overrides `name_attribute` parameter.

`name_attribute` parameter reference an attribute of an LDAP entry. If the
attribute is of type `distinguishedName`, you can specify the field of the DN to
use. e.g. `name.cn` targets the first `cn` RDN of the `name` attribute. If an
attribute is defined multiple times, one role is generated for each value.

`options`
define
[Postgres role options](https://www.postgresql.org/docs/current/static/sql-createrole.html).
Currently, only boolean options are supported. Namely: `BYPASSRLS`, `LOGIN`,
`CREATEDB`, `CREATEROLE`, `INHERIT`, `REPLICATION` and `SUPERUSER`. `options`
can be a SQL snippet like `SUPERUSER NOLOGIN`, a YAML list like `[LOGIN,
CREATEDB]` or a dict like `{LOGIN: yes, SUPERUSER: no}`.

`members_attribute` parameter behave the same way as `name_attribute`. It allows
you to read members of a Postgres role from LDAP attribute. If the attribute is
a list in LDAP, all entries are considered a member of each roles generated by
the entry. Note that members roles are **not** automatically added. You must
define a `role` rule for each member too, with their own options.

`parent` or `parents` define one or more parent role. This is the reverse
relation of `members`. Unlike `*_attribute` parameters, `parent` supports only
static values.


### `grant` parameters

Grant rule is a bit simpler than role rule. It tells `ldap2pg` to ensure a
particular role has one defined ACL granted. An ACL assignment is identified
by an ACL name, a database, a schema and a role.

`acl` key references an ACL by its name.

`database` allows to scope the grant to a database. By default, `database` is
inherited from the synchronization map. The special database name `__all__`
means **all** databases. `ldap2pg` will loop every databases in the cluster but
`template0` and apply the `grant` or `revoke` query on it.

In the same way, `schema` allows to scope the grant to one or more schema,
regardless of database. If `schema` is `__any__` or `null`, the `grant` or
`revoke` query will receive `None` as schema. If `schema` is `__all__`,
`ldap2pg` will loop all schema including `information_scema` and yield a revoke
or grant on each.

`role` or `roles` keys allow to specify statically one or more role to grant the
ACL to. `role` must be a string or a list of strings. Referenced roles must be
created in the cluster and won't be implicitly created.

`role_attribute` specifies how to fetch role name from LDAP entries. Just like
any `*_attribute` key, it accepts a `DN` attribute as well e.g: `name.cn`.

`role_match` is a pattern allowing you to limit the grant to roles whom name
matches `role_match`.

Here is a full example:

``` yaml
grant:
  acl: ddl
  database: appdb
  schema: __any__
  role_attribute: cn
  role_match: *_RW
```


### Overall sync map structure

Synchronization map structure is very lean and DRY. The goal is that you don't
have to tell more than you need, while controlling everything.

Actually, the simplest sync map is the following:

``` yaml
-  role: alice
```

Yep. This is enough for `ldap2pg` ! It's just a list with a single static
mapping. It tells `ldap2pg` to ensure the role `alice` is defined with `CREATE
USER` defaults in the cluster. For the sake of simplicity, we'll use only static
mapping in this section to explain the various structures of sync map.

Unlike roles, ACL are not cluster wide. Actually, Postgres allows you to define
ACL per columns in a table in a schema in a database. `ldap2pg` deal is to ease
to scope ACL up to schema. Deeper distinction is left to the user.

You can group mapping per database and per schema in the synchronization map,
using simple dictionnary. This avoids you to repeat `database` and `schema` key
in grant rules.

For example, here is the way to grant `connect` ACL to one database named
`appdb` to `alice` user:

``` yaml
sync_map:
  appdb:
    - grant:
        role: alice
        # database: appdb is implicit.
        acl: connect
```

If you want to grant an ACL to a specific schema, you could do this with:

``` yaml
sync_map:
  appdb:
    appschema:
      - grant:
          role: alice
          # database: appdb is implicit.
          # schema: appschema is implicit.
          acl: dml
```

When database is not defined, the mapping is assigned to the pseudo database
`__all__`:

``` yaml
sync_map:
- role: alice
```

Strictly equals to:

``` yaml
sync_map:
  __all__:
  - role: alice
```

When schema is not defined, the mapping is assigned to the pseudo schema
`__any__`. The previous sample strictly equals to:

``` yaml
sync_map:
  __all__:
    __any__:
    - role: alice
```

Beware that queries are not cached. If you copy a query from a database to
another, the LDAP search will be issued twice. As stated above, each `ldap:`
entry in the sync map will trigger **one** LDAP search, no less no more.
However, don't worry, `ldap2pg` deduplicates roles and grants produced by sync
map. You won't hit `ERROR: role alice already exists`.


### Sample

Here is an extended example of synchronization map:

``` yaml
acl_dict:
  connect:
    inspect: |
      WITH d AS (
          SELECT
              (aclexplode(datacl)).grantee AS grantee,
              (aclexplode(datacl)).privilege_type AS priv
          FROM pg_catalog.pg_database
          WHERE datname = current_database()
      )
      SELECT NULL as namespace, r.rolname
      FROM pg_catalog.pg_roles AS r
      JOIN d ON d.grantee = r.oid AND d.priv = 'CONNECT'
    grant: |
      GRANT CONNECT ON DATABASE {database} TO {role};
    revoke: |
      REVOKE CONNECT ON DATABASE {database} FROM {role}

sync_map:
- role:
    name: LDAP_USERS
    options: NOLOGIN
- ldap:
    base: cn=dba,ou=groups,dc=ldap2pg,dc=local
    filter: "(objectClass=groupOfNames)"
    attribute: member
  role:
    name_attribute: member.cn
    options: LOGIN SUPERUSER NOBYPASSRLS
    parent: LDAP_USERS
- ldap:
    base: ou=groups,dc=ldap2pg,dc=local
    filter: "(&(objectClass=groupOfNames)(cn=app*))"
    attributes: [cn, member]
  roles:
  - name_attribute: cn
    members_attribute: member.cn
    options: [NOLOGIN]
  - name_attribute: member.cn
    options:
      LOGIN: yes
    parents: [LDAP_USERS]
  grant:
    acl: connect
    role_attribute: member.cn
```

We provide a well
commented
[ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml), tested
on CI. This file is kept compatible with released `ldap2pg`. Feel free to start
with it and adapt it to your needs.

If you have trouble finding the right configuration for your needs,
please [file an issue](https://github.com/dalibo/ldap2pg/issues/new) to get
help.
