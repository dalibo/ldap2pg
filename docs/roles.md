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


## Custom inspection

By default, `ldap2pg` inspects all roles from Postgres and apply blacklist on
it. If you want `ldap2pg` to synchronsize only a subset of roles, you need to
customize inspection query in `postgres:roles_query`.

The columns returned by the row is: role name, array of member names, and a
special formatted value `{options}` containing role options columns
(rolcanlogin, rolsuper, etc.).

``` yaml
postgres:
  # Inspect only non SUPERUSER roles.
  roles_query: |
    SELECT role.rolname, array_agg(members.rolname) AS members, {options}
    FROM pg_catalog.pg_roles AS role
    LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
    LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
    WHERE role.rolsuper IS FALSE
    GROUP BY role.rolname, {options}
    ORDER BY 1;
```

You can also return only roles belonging to a `ldap_roles` group. You only have
to match the columns definition.


## Disable role management

You can tell `ldap2pg` to manage only ACL and never `CREATE` or `DROP` a role.
Set `postgres:roles_query` to `null` and never define a `role` rule in
`sync_map`.

``` yaml
postgres:
  roles_query: null

acls:
  ro: [__connect__, ...]

sync_map:
- grant:
    acl: ro
    database: mydb
    role: toto
```
