<h1>Managing Privileges</h1>

Managing privileges is tricky. ldap2pg tries to make this simpler and safer.


## Basics

The base design of ldap2pg is ambitious. Instead of revoke-everything-regrant
design, ldap2pg uses inspect-modify design. The process is the same as for
roles synchronization, including the three following steps:

1. Inspect Postgres cluster for granted privileges.
2. Loop `rules` and generate wanted grants set.
3. Compare the two sets of grants and update the Postgres cluster using
   `GRANT`, `REVOKE` and `ALTER DEFAULT PRIVILEGES`.

By default, ldap2pg does not manage privileges. To enable privileges
management, you must define at least one active privilege in [privileges]
section. The simplest way is to reuse [well-known privileges] shipped with
ldap2pg in an active group of privileges.

[privileges]: config.md#privileges
[well-known privileges]: wellknown.md

Defining a privilege triggers inspection of grants of this privilege in the
cluster and revocation of grants found in the cluster. To grant privileges and
keep grants found in the PostgreSQL cluster, you must use `grant` rule in
`rules`.

!!! warning "If it's not granted, revoke it!"

    Once a privilege is enabled, ldap2pg inspects the cluster and **revokes**
    all grants not required by a `grant` rule in `rules`.

ldap2pg inspects objects grants, owners and schemas only if configuration
defines privileges. ldap2pg inspects grants, owners and schemas in PostgreSQL
cluster **after** roles creation, update and drop.

In `ldap2pg.yml`, you specify privileges in a dictionnary named [privileges]
and grant them with [grant rules] in the `rules`.

[grant rules]: config.md#sync-map-grant

Inspecting privileges can cost a lot of resources. Also, revoking privileges is
known to be slow in PostgreSQL. The best practice is to grant privileges to a
group role and add user roles in the group to grant them all privileges at
once.

The following example defines three levels of privileges, the upper including
lower one. The `rules` defines three groups and grant the corresponding
privilege to the group:

``` yaml
privileges:
  reading:
  - __connect__
  - __usage_on_schemas__
  - __select_on_tables__

  writing:
  - reading  # include reading privileges
  - __insert_on_tables__
  - __update_on_tables__

  owning:
  - writing
  - __create_on_schemas__
  - __truncate_on_tables__

rules:
- role:
  - names:
    - readers
    - writers
    - owners
    options: NOLOGIN
- grant:
  - privilege: reading
    role: readers
  - privilege: writing
    role: writers
  - privilege: owning
    role: owners
```


## Managing public Privileges

PostgreSQL has a pseudo-role called `public`. It's a wildcard roles meaning
*every users*. All roles in PostgreSQL implicitly inherits from this `public`
role. Granting a privilege to `public` role grants to every role now and in the
future.

PostgreSQL also as the `public` schema. The `public` schema is a real schema
available in all databases.

PostgreSQL has some built-in privileges for `public` role. Especially for the
`public` schema. For example, `public` has `CONNECT` on all databases by
default. This means that you only rely on `pg_hba.conf` to configure access to
databases, which requires administrative access to the cluster and a
`pg_reload_conf()` call.

By default, ldap2pg includes `public` role in managed roles. [Well-known
privileges] knows how to inspect built-in privileges granted to `public` role
not explicitly revoked in the cluster.

If you want to preserve `public` role, rewrite [managed_roles_query] to not
include `public`.

[managed_roles_query]: config.md#postgres-managed-roles-query


## Managing Default Privileges

If you grant `SELECT` privileges on all tables in a schema to a role, this wont
apply to the new table created afterward. Instead of reexecuting ldap2pg after
the creation of every objects, PostgreSQL provides a way to define default
privileges.

PostgreSQL attaches default privileges to role. When the role creates an
object, PostgreSQL apply the corresponding default privileges to the new
object. The following default privileges ensure every new tables bob creates
will be selectable by alice: `ALTER DEFAULT PRIVILEGES FOR ROLE bob GRANT
SELECT ON TABLES TO alice;`.

If ldap2pg creates and drops owner roles, you want ldap2pg to configure
properly default privileges on these roles. If you hesitate to manage
privileges with ldap2pg, you should at least manage default privileges.

ldap2pg inspect the owners from PostgreSQL, not LDAP directory. There is no
`owner` or `for` parameter to `grant` rule in `rules`. Configuration file
defines owners either globally using [owners_query] or per schema using
[schemas_query].

[owners_query]: config.md#postgres-owners-query
[schemas_query]: config.md#postgres-schemas-query

The following example defines only default privileges and configure bob with
default privileges for alice.


``` yaml
postgres:
  owners_query: [bob]

privileges:
  reading:
  - __default_select_on_tables__

rules:
- roles:
    names:
    - alice
    - bob
    options: LOGIN
- grant:
    privilege: reading
    role: alice
```


## Defining Custom Privilege

[Well-known privileges] do not handle all cases. Sometime, you need ldap2pg to
manage a custom `GRANT` query. ldap2pg allows you to define custom privileges.

For ldap2pg, a privilege is a set of query: one to inspect the cluster, one to
grant the privilege and one to revoke it.

ldap2pg recognizes different kinds of privileges defined by the `type` parameter:

- `datacl` for `GRANT ON DATABASE`.
- `globaldefacl` for `ALTER DEFAULT PRIVILEGES` on a database. They are bound
  to an `owner`
- `nspacl` for `GRANT ON SCHEMA`. It's the default type.
- `defacl` for `ALTER DEFAULT PRIVILEGES IN SCHEMA`. They are bound to an
  `owner`.

Here is a full sample of custom privilege:

``` yaml
privileges:
  execute_myfunc:
    type: nspacl
    grant: GRANT EXECUTE ON FUNCTION {schema}.myfunc TO {role};
    revoke: REVOKE EXECUTE ON FUNCTION {schema}.myfunc TO {role};
    inspect: |
      WITH grants AS (
        SELECT
          pronamespace, proname, 
          (aclexplode(proacl)).grantee,
          (aclexplode(proacl)).privilege_type
        FROM pg_proc
      )
      SELECT
        nspname,
        pg_catalog.pg_get_userbyid(grantee) AS grantee,
      FROM grants
      JOIN pg_namespace ON pg_namespace.oid = pronamespace
      WHERE proname = 'myfunc' AND privilege_type = 'EXECUTE';

rules:
- grant:
    database: mydb
    schema: public
    privilege: execute_myfunc
    role: admin
```

`inspect` query is called **once** per database in the cluster to inspect
current grants of this privilege. If `null`, ldap2pg will consider this
privilege as never granted and will always re-grant. It's actually a bad idea
not to provide `inspect`. This won't allow ldap2pg to revoke privilege. Also,
this prevents you to check that a cluster is synchronized.

`inspect` query for `datacl` must return a rowset with two columns, the first is
unused, the second is the name of grantee.

`inspect` query for `nspacl` must return a rowset with three columns : the name
of the schema, the name of the grantee and a three state boolean called `full`.
`full` allows to manage `GRANT ON ALL TABLES IN SCHEMA`-like privilege.

If `full` is `t`, ldap2pg won't regrant. If `f`, ldap2pg will re-grant to
update the privilege or revoke to purge a partial grant.

If `full` is `NULL`, the privilege is considered unapplicable. ldap2pg will never
grant nor revoke this privilege. The main purpose of this case is to manage `ALL
TABLES IN SCHEMA` grants on schema with no tables.

`inspect` query for `defacl` must return a rowset with four columns : schema
name, grantee name, `full` state and owner name.

Writing `inspect` queries requires deep knowledge of Postgres internals. See
[System Catalogs](https://www.postgresql.org/docs/current/static/catalogs.html)
section in PostgreSQL documentation to see how privilege are actually stored in
Postgres. [Well-known privileges](wellknown.md) are a good starting point.

`grant` and `revoke` provide queries to respectively grant and revoke the
privilege. ldap2pg uses Python's [*Format String
Syntax*](https://docs.python.org/3.7/library/string.html#formatstrings) to
inject parameters in the query. The formatting accepts the the following
parameters:

- `database`: the database name. Strictly equals to `CURRENT_DATABASE`.
- `schema`: the schema name. Not available for `datacl` and `globaldefacl`
  privileges.
- `role`: the granted role name.
- `owner`: the role name of the object owner. Only available for default
privileges : `defacl` and `globaldefacl`.

If `grant` or `revoke` is `none`, ldap2pg will either skip grant or revoke on
the privilege and issue a warning. This mean you can write a revoke-only or a
grant-only privilege.
