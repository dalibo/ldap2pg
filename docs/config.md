<h1>Configuration</h1>

`ldap2pg` tries to be friendly regarding configuration. `ldap2pg` reads its
configuration from three sources, in the following order:

1. command line arguments.
2. environment variables.
3. configuration file.

The `--help` switch shows regular online documentation for CLI arguments. As of
version 1.0, this looks like:

``` console
$ ldap2pg --help
usage: ldap2pg [-c CONFIG] [-n] [-N] [-v] [--color] [--no-color] [-?] [-V]

Swiss-army knife to sync Postgres ACL from LDAP.

optional arguments:
  -c CONFIG, --config CONFIG
                        path to YAML configuration file (env: LDAP2PG_CONFIG)
  -n, --dry             do not touch Postgres, just print what to do (env: DRY)
  -N, --real            real mode, apply changes to Postgres (env: DRY)
  -v, --verbose         add debug messages including SQL and LDAP queries
                        (env: VERBOSE)
  --color               force color output (env: COLOR)
  --no-color            force plain text output (env: COLOR)
  -?, --help            show this help message and exit
  -V, --version         show version and exit

ldap2pg requires a configuration file to describe LDAP queries and role
mappings. See project home for further details. By default, ldap2pg runs in
dry mode.
```

Arguments can be defined multiple times. On conflict, the last argument is used.


# Environment variables

`ldap2pg` has no CLI switch to configure LDAP and Postgres connexion. However, `ldap2pg` supports `libpq` `PG*` env vars:

```
$ PGHOST=/var/run/postgresql PGUSER=postgres ldap2pg
Using /home/src/ldap2pg/ldap2pg.yml.
Starting ldap2pg 1.0a3.
Running in dry mode. Postgres will be untouched.
Inspecting Postgres...
...
```

See `psql(1)` for details on `libpq` env vars. `ldap2pg` also accept an extra
env var named `PGDSN` to define in
a
[`libpq` connection string](https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING):

```
$ PGDSN=postgres://postgres@localhost:5432/ ldap2pg
...
$ PGDSN=postgres://postgres@localhost:5432/ ldap2pg
...
```

There is no such standard with LDAP clients. `ldap2pg` adds a few variables that
mimic `libpq` behaviour:

- `LDAP_HOST`: the host, defaults to `localhost`.
- `LDAP_PORT`: the port, defaults to `389`.
- `LDAP_BIND`: the bind DN, like `cn=you,dc=entreprise,dc=fr`.
- `LDAP_PASSWORD`: the password.

Other environment variables are available and described in either `ldap2pg.yml`
sample or CLI help.


# `ldap2pg.yml`

`ldap2pg` requires a config file where the synchronisation map is described.
Everything can be configured from the YAML file: verbosity, real mode, LDAP and
Postgres credentials and synchronization map.

## File location

`ldap2pg` searches for files in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set, the standards file locations are
ignored.

``` console
$ ls *.yml
ldap2pg.yml
$ ldap2pg --config my-inexistant-file.yml
Cannot access configuration file my-inexistant-file.yml.
$
```

## YAML format

The first section contains various parameters for ldap2pg behaviour. The most
important is `dry` which tell whether `ldap2pg` runs in dry mode or real mode.
It\'s better to set real mode from CLI with `--real` argument.

``` yaml
# Colorization. env var: COLOR=<anything>
color: yes

# Verbose messages. Includes SQL and LDAP queries. env var: VERBOSE
verbose: no

# Dry mode. env var: DRY=<anything>
dry: yes
```

The next two sections define connexion parameters for LDAP and Postgres. Beware
that if `ldap2pg` refuses to read a password from a group readable or world
readable `ldap2pg.yml`.

``` yaml
ldap:
  host: ldap2pg.local
  port: 389
  bind: cn=admin,dc=ldap2pg,dc=local
  password: SECRET

postgres:
  dsn: postgres://user@%2Fvar%2Frun%2Fpostgresql:port/
```

## Synchronization map

The synchronization map is an ordered list of mapping. Each mapping is a dict
with a few known keys describing an LDAP query and a set of rules to generate
roles from entries returned by LDAP.

`ldap` define the LDAP query. `base`, `filter` and either `attribute` or
`attributes` must be defined. `attribute` can be either a string or a list. An
LDAP query is not mandatory. You need at least one attribute, an empty entry is
useless. `ldap2pg` can create roles defined statically from YAML.

`role` or `roles` contains one or more rules to generate Postgres role for each
entry returned by the LDAP search. A role can also be statically defined.

### `role` parameters

`name` or `names` contains one or more static name of role you want to define in
Postgres. This is usefull to e.g. define a `ldap_users` group. `names` parameter
overrides `name_attribute` parameter.

`name_attribute` parameter reference an attribute of an LDAP entry. If the
attribute is of type `distinguishedName`, you can specify the field of the DN to
use. e.g. `name.cn` targets the common name of the `name` attribute. If a
attribute is defined multiple times, a role is generated for each value.

`options`
define
[Postgres role options](https://www.postgresql.org/docs/current/static/sql-createrole.html).
Currently, only boolean options are supported. Namely: `BYPASSRLS`, `LOGIN`,
`CREATEDB`, `CREATEROLE`, `INHERIT`, `REPLICATION` and `SUPERUSER`. `options`
can be a SQL snippet like `SUPERUSER NOLOGIN`, a YAML list like `[LOGIN,
CREATEDB]` or a dict like `{LOGIN: yes, SUPERUSER: no}`.

`members_attribute` parameter behave the same way as `name_attribute`. It allows
you to read members of a Postgres role from then entries. If the attribute is a
list in LDAP, all entries are considered a member of each roles generated by the
entry. Note that members roles are **not** automatically added. You must define
a `role` rule for members too, with their own options.

`parent` or `parents` define one or more parent role. This is the reverse
relation of `members`. Unlike `*_attribute` parameters, `parent` supports only
static values.

Here is an extended example of synchronization map:

``` yaml
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
```
