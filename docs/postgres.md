<h1>Inspecting Postgres cluster</h1>

ldap2pg follows the explicit create / implicit drop and explicit grant / implicit revoke pattern.
Thus properly inspecting cluster for what you want to drop/revoke is very crucial to succeed in synchronization.

ldap2pg inspects databases, schemas, roles, owners and grants with SQL queries.
You can customize all these queries in the `postgres` YAML section
with parameters ending with `_query`.
See [ldap2pg.yaml reference] for details.

[ldap2pg.yaml reference]: config.md#postgres-parameters


## What databases to synchronize ?

`databases_query` returns the flat list of databases to manage.
The `databases_query` must return the default database as defined in `PGDATABASE`.
When dropping roles, ldap2pg loops the databases list to reassign objects and clean GRANTs of to be dropped role.
This databases list also narrows the scope of GRANTs inspection.
ldap2pg will revoke GRANTs only on these databases.
See [ldap2pg.yaml reference] for details.

``` yaml
postgres:
  databases_query: |
    SELECT datname
    FROM pg_catalog.pg_database
    WHERE datallowconn IS TRUE;
```


## Synchronize a subset of roles

By default, ldap2pg manages all roles from Postgres it has powers on, minus the default blacklist.
If you want ldap2pg to synchronsize only a subset of roles,
you need to customize inspection query in `postgres:managed_roles_query`.
The following query excludes superusers from synchronization.

``` yaml
postgres:
  managed_roles_query: |
    SELECT 'public'
    UNION
    SELECT rolname
    FROM pg_catalog.pg_roles
    WHERE rolsuper IS FALSE
    ORDER BY 1;
```

ldap2pg will only drop, revoke, grant on roles returned by this query.

A common case for this query is to return only members of a group like `ldap_roles`.
This way, ldap2pg is scoped to a subset of roles in the cluster.

The `public` role does not exists in the system catalog.
Thus if you want ldap2pg to manage `public` privileges,
you must include explicitly `public` in the set of managed roles.
This is the default.
Of course, even if `public` is managed, ldap2pg won't drop or alter it if it's not in the directory.

A safety net to completely ignore some roles is [roles_blacklist_query].

``` yaml
postgres:
  roles_blacklist_query: [postgres, pg_*]  # This is the default.
```

[roles_blacklist_query]: config.md#postgres-roles-blacklist-query

!!! note

    A pattern starting with a `*` **must** be quoted.
    Else you'll end up with a YAML error like `found undefined alias`.


## Inspecting Schemas

For schema-wide privileges, ldap2pg needs to known managed schemas for each database.
This is the purpose of `schemas_query`.

[schemas_query]: config.md#postgres-schemas-query


## Configuring owners default privileges

To configure default privileges, use the `default` keyword when referencing a privilege:

``` yaml
privileges:
  reading:
  - default: global
    type: SELECT
    on: TABLES
```

Then grant it using `grant` rule:

``` yaml
rules:
- grant:
  - privilege: reading
    role: readers
    schema: public
    owner: ownerrole
```

You can use `__auto__` as owner.
For each schema, ldap2pg will configure every managed role having `CREATE` privilege on schema.

``` yaml
rules:
- grant:
  - privilege: reading
    role: readers
    schema: public
    owner __auto__
```

ldap2pg configures default privileges last, after all effective privileges.
Thus `CREATE` on schema is granted before ldap2pg inspects creators on schemas.


## Static Queries

You can replace all queries with a **static list** in YAML. This list will be
used as if returned by Postgres. That's very handy to freeze a value like
databases or schemas.

``` yaml
postgres:
  databases_query: [postgres]
  schemas_query: [public]
```
