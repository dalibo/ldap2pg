<h1>Managing roles</h1>

`ldap2pg` synchronizes Postgres roles in three steps:

1. Inspect Postgres for existing roles, their options and their membership.
2. Loop `sync_map` and generate roles specs from `role` rules
3. Compare the two roles sets and apply to the Postgres cluster using `CREATE`,
   `DROP` and `ALTER`.


## `role` rule

A `role` rule is a dict or a list of dict. Here is a full sample:

``` yaml
sync_map:

# A single static roles
- role: ldap_users

# Two *group* roles
- role:
    names:
    - rw
    - ro
    options: NOLOGIN

# A supersuer, and user roles.
- roles:
  - name: tata
    options: LOGIN SUPERUSER
  - name: toto
    parent: rw
    options: LOGIN
  - name: titi
    parent: ro
    options: LOGIN
```


The `role` rule accepts the following keys:

`name_attribute` maps a LDAP entry attribute to a role name.

`name` or `names` contains one or more static name of role you want in Postgres.
This is useful to e.g. define a `ro` group. `names` parameter overrides
`name_attribute` parameter.

`options` statically defines [Postgres role
options](https://www.postgresql.org/docs/current/static/sql-createrole.html).
Currently, only boolean options are supported. Namely: `BYPASSRLS`, `LOGIN`,
`CREATEDB`, `CREATEROLE`, `INHERIT`, `REPLICATION` and `SUPERUSER`. `options`
can be a SQL snippet like `SUPERUSER NOLOGIN`, a YAML list like `[LOGIN,
NOCREATEDB]` or a dict like `{LOGIN: yes, SUPERUSER: no}`.

`members` or `members_attribute` define members of the Postgres role. Note that
members roles are **not** automatically created in Postres cluster. You must
define a `role` rule for each member too, with their own options.

`parent`, `parents` or `parents_attribute` define one or more parent role. It's
the reverse meaning of `members_attribute`.


## Ignoring roles

`ldap2pg` totally ignore roles matching one of the glob pattern defined in
`postgres:blacklist`:

``` yaml
postgres:
  # This is the default value.
  blacklist: [postgres, pg_*]
```

The role blacklist is also applied to grants. `ldap2pg` will never apply `GRANT`
or `REVOKE` on a role matching one of the blacklist patterns.


## Disable role management

You can tell `ldap2pg` to manage only privileges and never `CREATE` or `DROP` a
role. Set `postgres:roles_query` to `null` and never define a `role` rule in
`sync_map`.

``` yaml
postgres:
  roles_query: null

privileges:
  ro: [__connect__, ...]

sync_map:
- grant:
    privilege: ro
    database: mydb
    role: toto
```
