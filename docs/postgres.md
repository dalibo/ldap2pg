<h1>Inspecting Postgres cluster</h1>

`ldap2pg` follows the explicite create / implicit drop and explicit grant /
implicit revoke model. Thus properly inspecting cluster for what you want to
drop/revoke is very crucial to succeed in synchronisation.

`ldap2pg` inspects databases, schemas, roles, owners and grants with SQL
queries. You can customize all these queries in the `postgres` YAML section with
parameters ending with `_query`.


## What databases to synchronize ?

`databases_query` return the flat list of databases to manage. When dropping
roles, `ldap2pg` loop the databases list to reassign objects and clean GRANTs of
to be dropped role. This databases list also narrow the scope of GRANTs
inspection. `ldap2pg` will revoke GRANTs only on these databases. By default,
`ldap2pg` lists all databases with connection allowed. This includes
 `template1` and `postgres`.

```
postgres:
  databases_query: |
    SELECT datname
    FROM pg_catalog.pg_database
    WHERE datallowconn IS TRUE;
```


## Synchronize a subset of roles

By default, `ldap2pg` inspects all roles from Postgres. If you want `ldap2pg` to
synchronsize only a subset of roles, you need to customize inspection query in
`postgres:managed_roles_query`.

``` yaml
postgres:
  # Inspect only non SUPERUSER roles.
  managed_roles_query: |
    SELECT 'public'
    UNION
    SELECT rolname
    FROM pg_catalog.pg_roles
    WHERE rolsuper IS FALSE
    ORDER BY 1;
```

`ldap2pg` will only drop, revoke, grant on roles returned by this query. This
*whitelist* also applies to members. Only members matching this list may be
removed from a group. Members not matching this list will be left in the group.

A common case for this query is to return only members of a group like
`ldap_roles`. This case is tested in
[ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml) sample.
This way, `ldap2pg` is scoped to a subset of roles in the cluster.

The `public` role does not exists in the system catalog. Thus if you want
`ldap2pg` to manage `public` privileges, you must include explicitly `public` in
the set of managed roles. This is the default. Of course, even if `public` is
managed, `ldap2pg` won't drop or alter it if it's not in the directory.


A safety net to completely ignore some roles is available : `blacklist`.
`blacklist` is a list of `glob` patterns. Every roles matching one of
`blacklist` patterns will be totally ignored from roles and privileges
synchronisation.

``` yaml
postgres:
  # This is the default.
  blacklist: [postgres, pg_*]
```


!!! note

    A pattern starting with a `*` **must** be quoted. Else you'll end up with a
    YAML error like `found undefined alias`.


## Inspecting Schema & Owners

Except with database and global default privileges, almost all privileges are
schema aware. Thus `ldap2pg` needs to known what schemas are in each database.
This is the purpose of `schemas_query`.

When managing `ALTER DEFAULT PRIVILEGES` with `ldap2pg`, you must tell who are
owners. Owners are roles supposed to create or drop objects in database such as
tables, views, functions, etc. `ldap2pg` checks that every owners have proper
default privileges. This way you dont have to re-run `ldap2pg` after update on
your SQL schema to grant privileges.

There is two ways of listing owners: *globally* or *per schema*. With
`owners_query` you can specify a global list of owners common to all databases
and all schemas. With `schemas_query` you can specify owners *per schema*.

The `managed_roles` whitelist applies to owners from either `schemas_query` or
`owners_query`.


### Global Owners Example

This is the default configuration:

``` yaml
postgres:
  schemas_query: |
    SELECT nspname FROM pg_catalog.pg_namespace;
  owners_query: |
    SELECT role.rolname
    FROM pg_catalog.pg_roles AS role
    WHERE role.rolsuper IS TRUE;
```


### Per-Schema Owners Example

In this example, each schema is associated with a `owners_%` group whose name
includes schema name.

``` yaml
postgres:
  schemas_query: |
    SELECT
      nspname,
      array_agg(owner.rolname) FILTER (WHERE rolname IS NOT NULL)
    FROM pg_catalog.pg_namespace
    LEFT OUTER JOIN pg_catalog.pg_roles AS owners_group
      ON owners_group.rolname = 'owners_' || nspname
    LEFT OUTER JOIN pg_catalog.pg_auth_members AS ms ON ms.roleid = owners_group.oid
    LEFT OUTER JOIN pg_catalog.pg_roles AS owner ON owner.oid = ms.member
    GROUP BY 1
```

`schemas_query` is executed once per database. Thus, you could use
`schemas_query` to compute **per-database** owners:

``` yaml
postgres:
  schemas_query: |
    SELECT
      nspname,
      ARRAY['owner_' || current_database()]
    FROM pg_catalog.pg_namespace;
```


## Static Queries

You can replace all queries with a **static list** in YAML. This list will be
used as if returned by Postgres. That's very handy to freeze a value like
databases or schemas.

``` yaml
postgres:
  databases_query: [postgres]
```
