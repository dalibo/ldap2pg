<h1>Managing Privileges</h1>

Managing privileges is tricky. `ldap2pg` tries to make this simpler and safer.

The base design of `ldap2pg` is ambitious. First, it inspects Postgres cluster
for grants, then loops `sync_map` to determine what privileges should be
granted, then compares and applies a diff on the cluster with `ALTER DEFAULT
PRIVILEGES`, `GRANT` or `REVOKE` queries.

In `ldap2pg.yml`, you specify privileges in a dictionnary named `acls` and grant
them with `grant` rules in the `sync_map`:

```yaml
acls:
  myacl:
    type: nspacl
    grant: 'GRANT ALL ON LARGE OBJECTS IN SCHEMA {schema}'

  mygroup:
  - __all_on_tables__
  - __all_on_sequences__

sync_map:
- grant:
    acl: myacl
    role: admin
```

A privileges defined as a YAML list is a *group of priveleges*. A group can
include other groups.

`ldap2pg` ships an extensive set of [well-known privileges](wellknown.md), see
dedicated documentation page for them.


## Enabling Privilege

`ldap2pg` disables privileges whose name starts with `_` or `.` **unless**
included in an privilege group not starting with `_` or `.`. An enabled privileg
is a privilege whose name does not start with `_` or `.` and is not included in
an enabled privilege.

``` yaml
acls:
  _disabled:
    inspect: NEVER EXECUTED

  __included__:
    inspect: SELECT rolname ...
    revoke: REVOKE ... FROM {role};

  group:
  - __included__
  
sync_map:
- grant:
    acl: group
    role: myrole
```

!!! warning "If it's not granted, revoke it!"

    Once a privilege is enabled, `ldap2pg` inspects the cluster and **revokes**
    all grants not required by a `grant` rule described below.


## `grant` rule

In `sync_map`, you can grant privilege with the `grant` rule. Here is a full
sample:

``` yaml
acls:
  ro:
  - __connect__
  - __usage_on_schema__

sync_map:
- ldap:
    base: ...
  grant:
    database: appdb
    acl: ro
    schema: appns
    role_attribute: cn
    role_match: *_RO
```

Grant rule is a bit simpler than role rule. It tells `ldap2pg` to ensure a
particular role is granted one defined ACL. An ACL assignment is identified by a
privilege name, a database, a schema and a grantee role name.

`acl` key references a privilege by its name.

`database` allows to scope the grant to a database. The special database name
`__all__` means **all** databases. `__all__` is the default value. With
`__all__`, `ldap2pg` will loop every databases in the cluster where
`datallowconn` is set and apply the `grant` or `revoke` query on it.

In the same way, `schema` allows to scope the grant to one or more schema,
regardless of database. If `schema` is `__all__`, `ldap2pg` will loop all
schemas of the database and yield a revoke or grant on each. Some privileges are
schema independant, like `CONNECT`, they will be granted once from sync user's
database.

`role` or `roles` keys allow to specify statically one or more grantee name.
`role` must be a string or a list of strings. Referenced roles must be created
in the cluster and won't be implicitly created.

`role_attribute` maps an attribute from LDAP entries to a role name in the
GRANT. Just like any `*_attribute` key, it accepts a `DN` member as well e.g:
`distinguished_name.cn`.

`role_match` is a pattern allowing you to limit the grant to roles whom name
matches `role_match`.

There is no `revoke` rule. Any grant found in the cluster is revoked unless it's
explicitly granted with `grant` rule.


## Defining Custom Privilege

[Well-known privileges](wellknown.md) do not handle all cases. Sometime, you need
`ldap2pg` to manage a custom `GRANT` query. Adding custom privilege is quite easy.

For `ldap2pg`, a privilege is a set of query: one to inspect the cluster, one to
grant the privilege and one to revoke it.

`ldap2pg` recognize different kinds of privileges:

- `datacl` are for `GRANT ON DATABASE`.
- `globaldefacl` are `ALTER DEFAULT PRIVILEGES` on a database. They are bound to
  an `owner`
- `nspacl` are for `GRANT ON SCHEMA`. It's the default type.
- `defacl` are for `ALTER DEFAULT PRIVILEGES IN SCHEMA`. They are bound to an
  `owner`.

Here is a full sample of custom privilege:

``` yaml
acls:
  execute_myfunc:
    type: nspacl
    grant: GRANT EXECUTE ON {schema}.myfunc TO {role};
    revoke:  GRANT EXECUTE ON {schema}.myfunc TO {role};
    inspect: |
      WITH grants AS (
        SELECT
          pronamespace, proname, 
          (aclexplode(proacl)).grantee,
          (aclexplode(proacl)).privilege_type
        FROM pg_procs
      )
      SELECT
        nspname,
        pg_catalog.pg_get_userbyid(grantee),
      FROM grants
      JOIN pg_namespace ON pg_namespace.oid = pronamespace
      WHERE proname = 'myfunc' AND privilege_type = 'EXECUTE';

sync_map:
- grant:
    database: mydb
    schema: public
    acl: execut_myfunc
    role: admin
```

`inspect` query is called **once** per database in the cluster to inspect
current grants of this privilege. If `null`, `ldap2pg` will consider this
privilege as never granted and will always re-grant. It's actually a bad idea
not to provide `inspect`. This won't allow `ldap2pg` to revoke privilege. Also,
this prevents you to check that a cluster is synchronized.

`inspect` query for `datacl` must return a rowset with two columns, the first is
unused, the second is the name of grantee.

`inspect` query for `nspacl` must return a rowset with three columns : the name
of the schema, the name of the grantee and a three state boolean called `full`.
`full` allows to manage `GRANT ON ALL TABLES IN SCHEMA`-like privilege.

If `full` is `t`, `ldap2pg` won't regrant. If `f`, `ldap2pg` will re-grant to
update the privilege or revoke to purge a partial grant.

If `full` is `NULL`, the privilege is considered unapplicable. `ldap2pg` will never
grant nor revoke this privilege. The main purpose of this case is to manage `ALL
TABLES IN SCHEMA` grants on schema with no tables.

`inspect` query for `defacl` must return a rowset with four columns : schema
name, grantee name, `full` state and owner name.

Writing `inspect` queries requires deep knowledge of Postgres internals. See
[System Catalogs](https://www.postgresql.org/docs/current/static/catalogs.html)
section in PostgreSQL documentation to see how privilege are actually stored in
Postgres. [Well-known privileges](wellknown.md) are a good starting point.

`grant` and `revoke` provide queries to respectively grant and revoke the privilege.
The query is formatted with three parameters: `database`, `schema`, `role` and
`owner`. `database` strictly equals to `CURRENT_DATABASE`. `ldap2pg` uses
Python's [*Format String
Syntax*](https://docs.python.org/3.7/library/string.html#formatstrings). See
example below. In verbose mode, you will see the formatted queries.

If `none`, `ldap2pg` will either skip grant or revoke on the privilege and issue a
warning. This mean you can write a revoke-only or a grant-only privilege.
