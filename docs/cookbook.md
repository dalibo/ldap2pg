<h1>Cookbook</h1>

Here in this cookbook, you'll find some recipes for various use case of
`ldap2pg`.

If you struggle to find a way to setup `ldap2pg` for your needs, please [file an
issue](https://github.com/dalibo/ldap2pg/issues/new) so that we can update
*Cookbook* with new recipes ! Your contribution is welcome!


# Don't Synchronize Superusers

Say you don't want to manage superusers in the cluser with `ldap2pg`, just
regular users. E.g. you manage superusers through Ansible or another LDAP
directory. By default, `ldap2pg` will purge these users not in LDAP directory.

To avoid that, you can put all superusers in `postgres:blacklist` settings from
YAML file. The drawback is that you must keep it sync with the cluster.

Another option is to **customize the SQL query for roles inspection** with an
ad-hoc `WHERE` clause. Just as following.

``` yaml
postgres:
  roles_query: |
    SELECT
        role.rolname, array_agg(members.rolname) AS members,
        {options}
    FROM
        pg_catalog.pg_roles AS role
    LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
    LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
    WHERE role.rolsuper IS FALSE
    GROUP BY role.rolname, {options}
    ORDER BY 1;
```

This way `ldap2pg` will ignore all superusers defined in the cluster. You are
safe. This customization can be used for other case where you want to split
roles in different sets with different policies.

The query must return a set of row with the rolname as first column, an array
with the name of all members of the role as second column, followed by columns
defined in `{options}` template variable. `{options}` contains the ordered
columns of managed role options as supported by `ldap2pg`. `ldpa2pg` uses
Python's [*Format String
Syntax*](https://docs.python.org/3.7/library/string.html#formatstrings). Only
`options` substitution is available. `%` is safe.

# RO ACLs

Say you want to manage `GRANT SELECT` privileges based on LDAP directory. Here is an implementation of `inspect`, `grant` and `revoke` queries.

``` yaml
acl_dict:
  # SELECT on TABLES
  ro:
    inspect: |
      WITH
        def AS (
          SELECT
            nsp.oid, nsp.nspname,
            (aclexplode(defaclacl)).grantee,
            (aclexplode(defaclacl)).privilege_type
          FROM pg_catalog.pg_default_acl def
          JOIN pg_catalog.pg_namespace nsp ON nsp.oid = def.defaclnamespace
          WHERE defaclobjtype = 'r'
        ),
        nspacl AS (
          -- All namespace and role having grant on it, and array of available
          -- relations in the namespace.
          SELECT
            nsp.oid, nsp.nspname,
            (aclexplode(nspacl)).grantee,
            (aclexplode(nspacl)).privilege_type,
            ARRAY(SELECT UNNEST(array_agg(rel.relname)) ORDER BY 1) AS relname
          FROM pg_catalog.pg_namespace nsp
          LEFT OUTER JOIN pg_catalog.pg_class rel
            ON rel.relnamespace = nsp.oid AND relkind IN ('r', 'v')
          WHERE nsp.nspname NOT LIKE 'pg_%'
          GROUP BY 1, 2, 3, 4
        ),
        rel AS (
          -- All namespace, role and relation privilege granted to what relation
          -- in the namespace.
          SELECT
            table_schema as "schema",
            grantee, privilege_type,
            -- Aggregate the relation grant for this privilege.
            ARRAY(SELECT UNNEST(array_agg(table_name::name)) ORDER BY 1) as tables
          FROM information_schema.role_table_grants
          GROUP BY 1, 2, 3
        )

      -- Now list all users per schema who have USAGE or SELECT to any
      -- relations or have SELECT default privilege.
      SELECT
        nsp.nspname,
        rol.rolname,
        (
          nspacl.oid IS NOT NULL AND
          def.oid IS NOT NULL AND
          -- Here, compare arrays to ensure SELECT grant is for all relation in
          -- namespace.
          coalesce(select_.tables, ARRAY[NULL]::name[]) = nspacl.relname
        )
      -- First, produce all combination of roles and namespace in database.
      FROM pg_catalog.pg_roles rol
      CROSS JOIN pg_catalog.pg_namespace nsp
      -- inspect any schema privileges
      LEFT OUTER JOIN nspacl
        ON nspacl.grantee = rol.oid AND
           nspacl.oid = nsp.oid
      -- inspect default privileges on schema
      LEFT OUTER JOIN def
        ON def.grantee = rol.oid AND
           def.privilege_type = 'SELECT' AND
           def.oid = nsp.oid
      -- inspect tables privileges in schema.
      LEFT OUTER JOIN rel AS select_
        ON select_.privilege_type = 'SELECT' AND
           select_."schema" = nsp.nspname AND
           select_.grantee = rol.rolname
      -- filter only if at lease one privileges is granted.
      WHERE nspacl.oid IS NOT NULL OR def.oid IS NOT NULL OR select_."schema" IS NOT NULL
      ORDER BY 1, 2;
    grant: |
      GRANT USAGE ON SCHEMA {schema} TO {role};
      GRANT SELECT ON ALL TABLES IN SCHEMA {schema} TO {role};
      ALTER DEFAULT PRIVILEGES IN SCHEMA {schema} GRANT SELECT ON TABLES TO {role};
    revoke: |
      ALTER DEFAULT PRIVILEGES IN SCHEMA {schema} REVOKE SELECT ON TABLES FROM {role};
      REVOKE SELECT ON ALL TABLES IN SCHEMA {schema} FROM {role};
      REVOKE USAGE ON SCHEMA {schema} FROM {role};

sync_map:
- grant:
    role: daniel
    acl: ro
    database: frontend
    schema: __all__
```

As you can see, the inspect query is quite tricky. The complexity come from the
aggregation of multiple `GRANT` into a single ACL. Also, `GRANTO ON ALL TABLES`
registers several ACL that must be checked. You can adapt this ACL to manage
other privileges like `INSERT`, `UPDATE` and make a `rw` ACL alike.


# Synchronize only ACL

You may want to trigger `GRANT` and `REVOKE` without touching roles. e.g. you
update privileges after a schema upgrade.

To do this, create a distinct configuration file. You must first disable roles
introspection, so that `ldap2pg` will never try to drop a role. Then you must
ban any `role` rule from the file. You can still trigger LDAP searches to
determine to which role you want to grant an ACL.

``` yaml
# File `ldap2pg.acl.yml`

postgres:
  # Disable roles introspection by setting query to null
  roles_query: null

acl_dict:
  rw: {}  # here define your ACLs

sync_map:
- ldap:
    base: cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker
    filter: "(objectClass=groupOfNames)"
    scope: sub
    attribute: member
  grant:
    role_attribute: member
    acl: rw
```
