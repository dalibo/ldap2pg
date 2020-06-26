<!-- markdownlint-disable MD033 MD041 MD046 -->

<h1><tt>ldap2pg.yml</tt></h1>

`ldap2pg` accepts a YAML configuration file usually named `ldap2pg.yml` and put
in working directory. Everything can be configured from the YAML file:
verbosity, LDAP and Postgres credentials, LDAP queries, privileges and
mappings.

!!! warning

    `ldap2pg` **requires** a config file where the synchronization map
    is described.


## File Location

`ldap2pg` searches for files in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set, `ldap2pg` skip searching the standard
file locations. You can specify `-` to read configuration from standard input.
This is helpful to feed `ldap2pg` with dynamic configuration.


## File Structure & Example

`ldap2pg.yml` is split in several sections :

- `postgres` : setup Postgres connexion and inspection queries.
- `ldap` : setup LDAP connexion.
- `privileges` : the definition of privileges.
- `sync_map` : the list of LDAP queries and associated mapping to roles and
  grants.
- finally some global parameters (verbosity, etc.).

We provide a simple well commented
[ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml), tested
on CI. If you don't know how to begin, it can be a good starting point.

!!! note

    If you have trouble finding the right configuration for your needs, feel free to
    [file an issue](https://github.com/dalibo/ldap2pg/issues/new) to get help.


## About YAML

YAML is a super-set of JSON. A JSON document is a valid YAML document. YAML very
permissive format where indentation is meaningful. See [this YAML
cheatsheet](https://medium.com/@kenichishibata/yaml-to-json-cheatsheet-c3ac3ef519b8)
for some example.


## Postgres Parameters

The `postgres` section defines connection parameters and SQL queries for
Postgres inspection.

The `postgres` section contains several `*_query` parameters. These parameters
can be either a string containing an SQL query or a YAML list to return a
static list of values.


### `dsn`

Specifies a PostgreSQL connexion URI.

``` yaml
postgres:
  dsn: postgres://user@%2Fvar%2Frun%2Fpostgresql:port/
```

!!! warning

    `ldap2pg` refuses to read a password from a group readable or world
    readable `ldap2pg.yml`.


### `databases_query`

The SQL query to list databases in the cluster. This defaults to all databases
connectable, thus including `template1`. You can override this with a YAML
list like other queries.

``` yaml
postgres:
  databases_query: "SELECT datname FROM pg_catalog.pg_databases;"
  # OR
  databases_query: [mydb]
```


### `managed_roles_query`

The SQL query to list the name of managed roles. ldap2pg restricts role
deletion and privilege edition to managed roles. Usualy, this query returns
children of a dedicated group like `ldap_roles`. By default, all roles found
are managed.

`public` is a special builtin role in Postgres. If `managed_roles_query`
returns `public` role in the list, ldap2pg will manage privileges on `public`.

``` yaml
postgres:
  managed_roles_query: |
    SELECT 'public'
    UNION
    SELECT DISTINCT role.rolname
    FROM pg_roles AS role
    LEFT OUTER JOIN pg_auth_members AS ms ON ms.member = role.oid
    LEFT OUTER JOIN pg_roles AS ldap_roles
      ON ldap_roles.rolname = 'ldap_roles' AND ldap_roles.oid = ms.roleid
    WHERE role.rolname = 'ldap_roles' OR ldap_roles.oid IS NOT NULL
    ORDER BY 1;
```


### `owners_query`

The SQL query to global list the names of object owners. ldap2pg execute this
query *once*, after all roles are created, before granting and revoking
privileges. You need this query only if you manage default privileges with
ldap2pg.

``` yaml
postgres:
  owners_query: |
    SELECT role.rolname
    FROM pg_catalog.pg_roles AS role
    WHERE role.rolsuper IS TRUE;
```

You can declare per-schema owners with `schemas_query`. However, unlike
`owners_query`, `schemas_query` is executed *before* creating users.


### `roles_blacklist_query`

The SQL query returning name and glob pattern to blacklist role from
management. ldap2pg won't touch anything on these roles.

``` yaml
postgres:
  roles_blacklist_query:
  - postgres
  - pg_*
  - rds_*
  - "rds*admin"
```

Beware that `*suffix` is a YAML reference. You must quote pattern beginning
with `*`.


### `roles_query`

The SQL query returning all roles, their options and their members. It's not
very useful to customize this. Prefer configure `roles_blacklist_query` and
`managed_roles_query` to reduce synchronization to a subset of roles.

Role's options varies from one PostgreSQL version to another. ldap2pg handle
this by injecting options columns in `{options}` substitution.

``` yaml
postgres:
  roles_query: |
    SELECT
        role.rolname, array_agg(members.rolname) AS members,
        {options},
        pg_catalog.shobj_description(role.oid, 'pg_authid') as comment
    FROM
        pg_catalog.pg_roles AS role
    LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
    LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
    GROUP BY role.rolname, {options}, comment
    ORDER BY 1;
```


### `schemas_query`

The SQL query returning the name of schemas in a database. ldap2pg execute this
query on each databases returned by `databases_query`. ldap2pg loops on objects
in theses schemas when inspecting GRANTs in the cluster.

``` yaml
postgres:
  schemas_query: |
    SELECT nspname FROM pg_catalog.pg_namespace
```


## LDAP Parameters

``` yaml
ldap:
  uri: ldap://ldap2pg.local:389
  binddn: cn=admin,dc=ldap2pg,dc=local
  user: saslusername
  password: SECRET
```


## `sync_map`

The synchronization map is a YAML list. We call each item a *mapping*. Three
sections compose a mapping:

- A `description` entry with a string logged before this mapping is processed.
- A `ldap` section describing a LDAP query.
- A `role` or `roles` section describing on or more rules to create [Postgres
  role](https://www.postgresql.org/docs/current/static/user-manag.html) from
  LDAP entries.
- A `grant` section describing on or more grant from LDAP entries.

`ldap` entry is optional, however either one of `roles` or `grant` is required.

!!! tip

    Defining the right sync map can be tedious. Start with is simple
    sync map to setup Postgres and LDAP connexion first and then define detailed
    synchronisation steps. Here is the simplest sync map:

    <pre class="highlight"><code class="language-yaml">sync_map:
    - role: toto
    </code></pre>

    It just means you want a role named `toto` in the cluster.


## Shortcuts

If the file is a YAML list, `ldap2pg` puts the list as `sync_map`. The two
following configurations are strictly equivalent:

``` console
$ ldap2pg -c -
- admin
$ ldap2pg -c -
sync_map:
- roles:
  - names:
    - admin
$
```

`database`, `schema`, `role`, `name`, `parent` and `member` can be either a
string or a list of strings. These keys have plural aliases, respectively
`databases`, `schema`, `roles`, `names`, `parents` and `members`.

<!-- Local Variables: -->
<!-- ispell-dictionary: "american" -->
<!-- End: -->
