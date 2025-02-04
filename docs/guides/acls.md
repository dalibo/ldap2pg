<h1>Custom ACL</h1>

ldap2pg comes with builtin ACLs for common objects like `DATABASE`, `SCHEMA`, `TABLE`, `FUNCTION`, etc.
PostgreSQL has a lot of other objects like `FOREIGN DATA WRAPPER`, `FOREIGN SERVER`, `FOREIGN TABLE`, `TYPE`, etc.
You may also want to manage custom ACL or something else.
Writing a custom ACL should help you get the job done.

!!! note

    Writing a custom ACL is quite advanced.
    You should be familiar with PostgreSQL ACLs and ldap2pg configuration.
    Ensure you have read [privileges] and [config] documentation,
    have a good understanding of [PostgreSQL ACLs]
    and have successfully synchronized privileges with ldap2pg.


Define custom ACL in YAML configuration file.

## Use case

Say you have a custom enum type `myenum` in your database.

``` sql
CREATE TYPE public.myenum AS ENUM ('toto', 'titi', 'tata');
```

We want to manage privileges on this object,
eventually other types,
with a custom ACL.


## Naming

Name your ACL after the keyword in `GRANT` or `REVOKE` statement.
From `GRANT USAGE ON TYPE mytype TO myrole`, name your ACL `TYPE`.


```yaml
acls:
  TYPE:
    ...
```


## Scope

PostgreSQL defines user types per schema.
Since we'll hardcode the schema,
we will scope our ACL to database.

``` yaml
acls:
  TYPE:
    scope: database
```


## Grant and Revoke

Writing GRANT and REVOKE statements is the easiest part.
See [grant](../config.md#acls-grant) for details on query format.
We'll use the `object` field of grant to store the name of the type.
public schema is hardcoded, as explained above.

```yaml
acls:
  TYPE:
    scope: database
    grant: GRANT <privilege> ON TYPE public.<object> TO <grantee>;
    revoke: REVOKE <privilege> ON TYPE public.<object> FROM <grantee>;
```

## Inspect

The inspect query is the most difficult part.
You need to master `aclitem`, `aclexplode` and `acldefault` PostgreSQL system builtins.
The signature of the inspect query depends on the scope of the ACL.

For `instance` scope:

- `type`: a string describing privilege type as SQL keyword.
- `object`: a string describing object name as SQL identifier.
- `grantee`: a string describing role name as SQL identifier.

For `database` scope:

- `type`: a string describing privilege type as SQL keyword.
- `object`: a string describing object name as SQL identifier.
- `grantee`: a string describing role name as SQL identifier.
- `partial`: a boolean indicating if the grant is partial.

partial tells ldap2pg to re-grant `ALL ... IN SCHEMA` privileges.
Since our ACL is handling one object at a time, `partial` will always be `false`.

ldap2pg sends a single parameter to inspect query: the effective list of privilege types managed by the configuration.
This list is an array of text.
ldap2pg expects query to filter other privileges out of the list.

For `TYPE` ACL, we will inspect privileges on `pg_type` system catalog.

``` yaml
acls:
  TYPE:
    scope: database
    grant: GRANT <privilege> ON <acl> public.<object> TO <grantee>;
    revoke: REVOKE <privilege> ON <acl> public.<object> FROM <grantee>;
    inspect: |
      WITH grants AS (
        SELECT typname,
               (aclexplode(COALESCE(typacl, acldefault('T', typowner)))).privilege_type AS priv,
               (aclexplode(COALESCE(typacl, acldefault('T', typowner)))).grantee::regrole::text AS grantee
          FROM pg_catalog.pg_type
        WHERE typnamespace::regnamespace = 'public'::regnamespace
          AND typtype <> 'b'  -- exclude base type.
      )
      SELECT grants.priv AS "privilege",
            grants.typname AS "object",
            CASE grants.grantee WHEN '-' THEN 'public' ELSE grants.grantee END AS grantee,
            FALSE AS partial
        FROM grants
      WHERE "priv" = ANY ($1)
      ORDER BY 2, 3, 1
      ;
```

As you see, it's not an easy query.
This query works on PostgreSQL 17.


## Using

You can now use your custom ACL in a profile.
Reference object in profile, not in grant rule.

```yaml
privileges:
  custom:
  - type: USAGE
    on: TYPE
    object: myenum

rules:
- roles:
    names:
    - alice
  grant:
    privilege: custom
    role: alice
```

You must reference all types manually in the profile.
Executing ldap2pg should produce changes in your database:

``` console
$ ldap2pg
...
16:52:02 CHANGE Would Revoke privileges.                         grant="USAGE ON TYPE myenum TO public" database=db0
16:52:02 CHANGE Would Grant privileges.                          grant="USAGE ON TYPE myenum TO alice" database=db0
16:52:02 INFO   Comparison complete.                             searches=0 roles=1 queries=5 grants=1
16:52:02 INFO   Use --real option to apply changes.
16:52:02 INFO   Done.                                            elapsed=44.345229ms mempeak=1.6MiB ldap=0s inspect=28.992071ms sync=0s
```

That's it.


## Debugging

If you encounter problem, isolate the issue.
Reduce the configuration to your custom ACL.
Synchronize a single role, grant to it.
Avoid LDAP searches, use only static rules.
Synchronize a single database.
Enable debug messages with `--verbose` option.

ldap2pg works database per database then ACL per ACL.
The messages for each ACL are as follow:

First line about your ACL has `acl=TYPE` record attribute.

```
17:13:35 DEBUG  Inspecting grants.                               acl=TYPE scope=database database=db0
```

Then you have messages for inspection: query and arguments.
For each grant returned, a `Found grant.` message appears.

```
17:13:35 DEBUG  Executing SQL query:
WITH grants AS (
...
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 1
;
 arg=[USAGE]
17:13:35 DEBUG  Found grant in Postgres instance.                grant="USAGE ON TYPE myenum TO public" database=db0
```

Then, ldap2pg expands grants generated by rule.
For each grant generated, a `Wants grant.` message is printed.

```
17:13:35 DEBUG  Wants grant.                                     grant="USAGE ON TYPE myenum TO alice" database=db0
```

Finally, ldap2pg prints changes it would apply.

```
17:13:35 CHANGE Would Revoke privileges.                         grant="USAGE ON TYPE myenum TO public" database=db0
17:13:35 DEBUG  Would Execute SQL query:
REVOKE USAGE ON TYPE public."myenum" FROM "public";
17:13:35 CHANGE Would Grant privileges.                          grant="USAGE ON TYPE myenum TO alice" database=db0
17:13:35 DEBUG  Would Execute SQL query:
GRANT USAGE ON TYPE public."myenum" TO "alice";
```

At the end, ldap2pg prints a conclusion message, even if no changes are required.

```
17:13:35 DEBUG  Privileges synchronized.                         acl=TYPE database=db0
```
