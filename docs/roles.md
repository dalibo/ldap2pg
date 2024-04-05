<h1>Managing roles</h1>

ldap2pg synchronizes Postgres roles in three steps:

1. Loop `rules` and generate wanted roles list from `role` rules.
2. Inspect Postgres for existing roles, their options and their membership.
3. Compare the two roles sets and apply to the Postgres cluster using `CREATE`,
   `DROP` and `ALTER`.

Each [role] entry in `rules` is a rule to generate zero or more roles with the corresponding parameters.
A `role` rule is like a template.
`role` rules allows to deduplicate membership and options by setting a list of names.

You can mix static rules and dynamic rules in the same file.

[role]: config.md#rules-role


## Running unprivileged

ldap2pg is designed to run unprivileged.
Synchronization user needs `CREATEROLE` option to manage other unprivileged roles.
`CREATEDB` options allows synchronization user to managed database owners.

ldap2pg user must have `createrole_self_grant` set to `inherit,set` to properly handle groups.

``` sql
CREATE ROLE ldap2pg LOGIN CREATEDB CREATEROLE;
ALTER ROLE ldap2pg SET createrole_self_grant TO 'inherit,set;
```

Running unprivileged before Postgres 16 is actually flawed.
You'd better just run ldap2pg with superuser privileges, you wont feel falsly secured.


## Ignoring roles

ldap2pg totally ignores roles matching one of the glob pattern defined in [roles_blacklist_query]:

``` yaml
postgres:
  # This is the default value.
  roles_blacklist_query: [postgres, pg_*]
```

The role blacklist is also applied to grants.
ldap2pg will never apply `GRANT` or `REVOKE` on a role matching one of the blacklist patterns.

[roles_blacklist_query]: config.md#postgres-roles-blacklist-query

ldap2pg blacklists its running user.


## Membership

ldap2pg manages parents of roles.
ldap2pg applies [roles_blacklist_query] to parents.
However, ldap2pg grants unmanaged parents.
This way, you can create a group manually and manages its members using ldap2pg.
