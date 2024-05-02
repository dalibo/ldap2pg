<h1>Managing Privileges</h1>

Managing privileges is tricky.
ldap2pg tries to make this simpler and safer.


## Basics

The base design of ldap2pg is ambitious.
Instead of revoke-everything-regrant design,
ldap2pg uses inspect-modify design.
The process is the same as for roles synchronization, including the three following steps:

1. Loop `rules` and generate wanted grants set.
2. Inspect Postgres cluster for granted privileges.
3. Compare the two sets of grants and update the Postgres cluster using
   `GRANT`, `REVOKE`.

ldap2pg synchronizes privileges one at a time, database by database.
ldap2pg synchronizes default privileges last.

By default, ldap2pg does not manage any privileges.
To enable privilege management, you must define at least one active privilege profile in [privileges] section.
The simplest way is to reuse [builtin privilege profiless] shipped with ldap2pg in an active group of privileges.

[privileges]: config.md#privileges
[builtin privilege profiles]: builtins.md


## Defining a Privilege Profile

A privilege profile is a list of references to either a privilege type on an ACL or another profile.
ldap2pg ships several predefined ACL like `DATABASE`, `LANGUAGE`, etc.
A privilege type is `USAGE`, `CONNECT` and so on as describe in as documented in PostgreSQL documentation, [section 5.7].
See [privileges] YAML section documentation for details on privilege profile format.

[section 5.7]: https://www.postgresql.org/docs/current/ddl-priv.html
[privileges]: config.md#privileges

ldap2pg loads referenced ACL by inspecting PostgreSQL cluster with carefully crafted queries.
An unreferenced ACL is ignored.
Inspected grants are supposed to revokation unless explicitly wanted by a `grant` rule.

!!! warning "If it's not granted, revoke it!"

    Once a privilege is enabled,
    ldap2pg **revokes** all grants found in Postgres instance and not required by a `grant` rule in `rules`.


## Extended Intance inspection

When managing privileges, ldap2pg has deeper inspection of Postgres instance.
ldap2pg inspects schemas after roles synchronization and before synchronizing privileges.
ldap2pg inspects objects owner after privileges synchronization and before synchronizing default privileges.
An object owner is a role having `CREATE` privilege on a schema.


## Granting Privilege Profile

[grant rules]: config.md#rules-grant

Inspecting privileges may consume a lot of resources on PostgreSQL instance.
Revoking privileges is known to be slow in PostgreSQL.
The best practice is to grant privileges to a group role and let user inherit privileges.
With ldap2pg, you can define static groups in YAML and inherit them when creating roles from directory.

Use [grant rule] to grant a privilege profile to one or more roles.
When granting privileges, you must define the grantee.
You may scope the grant to one or more databases, one or more schemas.
If the privilege profile includes default privileges, you may define the owners on which to configure default privileges.

By default, a grant applies to all managed databases as returned by [databases\_query],
to all schema of each database as returned by [schemas\_query].

[databases\_query]: config.md#postgres-databases-query
[schemas\_query]: config.md#postgres-schemas-query

## Example

The following example defines three privileges profile.
The `rules` defines three groups and grant the corresponding privilege profile:

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

Another way of including reading profile in writing is to writers group to inherit readers group.


## Managing public Privileges

PostgreSQL has a pseudo-role called `public`.
It's a wildcard roles meaning *every users*.
All roles in PostgreSQL implicitly inherits from this `public` role.
Granting a privilege to `public` role grants to every role now and in the future.

PostgreSQL also as the `public` schema.
The `public` schema is a real schema available in all databases.

PostgreSQL has some built-in privileges for `public` role.
Especially for the `public` schema.
For example, `public` has `CONNECT` on all databases by default.
This means that you only rely on `pg_hba.conf` to configure access to databases,
which requires administrative access to the cluster and a `pg_reload_conf()` call.

By default, ldap2pg includes `public` role in managed roles.
Predefined ACL knows how to inspect built-in privileges granted to `public`.
If you want to preserve `public` role, rewrite [managed_roles_query] to not include `public`.

[managed_roles_query]: config.md#postgres-managed-roles-query


## Managing Default Privileges

If you grant `SELECT` privileges on all tables in a schema to a role, this wont apply to new tables created afterward.
Instead of reexecuting ldap2pg after the creation of every objects,
PostgreSQL provides a way to define default privileges for future objects.

PostgreSQL attaches default privileges to the creator role.
When the role creates an object, PostgreSQL apply the corresponding default privileges to the new object.
e.g. `ALTER DEFAULT PRIVILEGES FOR ROLE bob GRANT SELECT ON TABLES TO alice;`
ensures every new tables bob creates will be selectable by alice:

If ldap2pg creates and drops creator roles, you want ldap2pg to properly configure default privileges on these roles.
If you wonder whether to manage privileges with ldap2pg, you should at least manage default privileges along creator.

ldap2pg inspects the creators from PostgreSQL, per schemas, not LDAP directory.
A creator is a role with LOGIN option and CREATE privilege on a schema.
You can manually set the target owner of a grant to any managed roles.

ldap2pg does not configure privileges on `__all__` schemas.
You are supposed to use `global` scope instead.
If you want to grant/revoke default privilege per schema, you must reference `schema` default.

The following example configures default privileges for alice to allow bob to SELECT on future tables created by alice.


``` yaml
privileges:
  reading:
  - default: global
    type: SELECT
    on: TABLES
  owning:
  - type: CREATE
    on: SCHEMAS

rules:
- roles:
    names:
    - alice
    - bob
    options: LOGIN
- grant:
    privilege: owning
    role: alice
- grant:
    privilege: reading
    role: bob
```


PostgreSQL has hard-wire global default privileges.
If a role does not have global default privileges configured, PostgreSQL assume some defaults.
By default, PostgreSQL just grant privileges on owner.
You can see them once you modify the default privileges.
PostgreSQL will copy the hard-wired values along your granted privileges.

If you don't explicitly re-grant these privileges in ldap2pg.yml,
ldap2pg will revoke these hard-wired privileges.
Actually, an owner of table don't need to be granted SELECT on its own tables.
Thus, the hard-wired defaults are useless.
You can let ldap2pg purge these useless defaults.
