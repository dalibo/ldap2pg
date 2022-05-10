<h1>Managing roles</h1>

ldap2pg synchronizes Postgres roles in three steps:

1. Inspect Postgres for existing roles, their options and their membership.
2. Loop `sync_map` and generate wanted roles list from `role` rules.
3. Compare the two roles sets and apply to the Postgres cluster using `CREATE`,
   `DROP` and `ALTER`.

Each [role] entry in `sync_map` is a rule to generate zero or more roles with
the corresponding parameters. A `role` rule is like a template. `role` rules
allows to deduplicate membership and options by setting a list of names.

You can mix static roles and roles generated with LDAP attributes in the same
file.

[role]: config.md#sync-map-role


## Ignoring roles

ldap2pg totally ignores roles matching one of the glob pattern defined in
[roles_blacklist_query]:

``` yaml
postgres:
  # This is the default value.
  blacklist: [postgres, pg_*]
```

The role blacklist is also applied to grants. ldap2pg will never apply `GRANT`
or `REVOKE` on a role matching one of the blacklist patterns.

[roles_blacklist_query]: config.md#postgres-roles-blacklist-query

ldap2pg never drop its connecting role.


## Disabling Role Management

You can tell ldap2pg to manage only privileges and never `CREATE` or `DROP` a
role. Set [roles_query] to `null` and never define a `role` rule in `sync_map`.

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

[roles_query]: config.md#postgres-roles-query
