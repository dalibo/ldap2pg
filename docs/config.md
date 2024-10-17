<!-- markdownlint-disable MD033 MD041 MD046 -->

<h1>ldap2pg.yml file reference</h1>

ldap2pg requires a YAML configuration file
usually named `ldap2pg.yml` and put in working directory.
Everything can be configured from the YAML file: Postgres inspect queries, LDAP searches, privileges and synchronization map.

!!! warning

    ldap2pg **requires** a config file where the synchronization map is described.


## File Location

ldap2pg searches for configuration file in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.
4. `/etc/ldap2pg/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set,
ldap2pg skips searching the standard file locations.
You can specify `-` to read configuration from standard input.
This is helpful to feed ldap2pg with dynamic configuration.


## File Structure

`ldap2pg.yml` is split in several sections :

- `postgres` : setup Postgres connexion and inspection queries.
- `privileges` : the definition of privileges profiles.
- `rules` : the list of LDAP searches and associated mapping to roles and
  grants.

The project provides a simple well commented [ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml),
tested on CI.
If you don't know how to begin, it is a good starting point.

!!! note

    If you have trouble finding the right configuration for your needs, feel free to
    [file an issue](https://github.com/dalibo/ldap2pg/issues/new) to get help.


### About YAML

YAML is a super-set of JSON.
A JSON document is a valid YAML document.
YAML is a very permissive format where indentation is meaningful.
See [this YAML cheatsheet](https://medium.com/@kenichishibata/yaml-to-json-cheatsheet-c3ac3ef519b8) for some example.

In `ldap2pg.yaml` file, you will likely use wildcard for glob pattern and curly brace for LDAP attribute injection.
Take care of protecting these characters with quotes.


## Postgres Parameters

The `postgres` section defines custom SQL queries for Postgres inspection.

The `postgres` section contains several `*_query` parameters.
These parameters can be either a string containing an SQL query
or a YAML list to return a static list of values,
skipping execution of a query on PostgreSQL cluster.


### `databases_query`  { #postgres-databases-query }

[databases_query]: #postgres-databases-query

The SQL query to list databases names in the cluster.
By default, ldap2pg searches databases it cans connect to and it can reassign objects to its owner.
ldap2pg loops databases to reassign objects before dropping a role.
ldap2pg manages privilege on each database.

``` yaml
postgres:
  databases_query: "SELECT datname FROM pg_catalog.pg_databases;"
  # OR
  databases_query: [mydb]
```

!!! note

    Configuring a _query parameter with a YAML list skip querying the cluster
    for inspection and forces ldap2pg to use a static value.


### `fallback_owner`  { #postgres-fallback-owner }

Name of the role accepting ownership of database of dropped role.

Before dropping a role, ldap2pg reassigns objects and purges ACL.
ldap2pg starts by reassigning database owner by the targetted user.
The new owner of the database is the *fallback owner*.
Other objects are reassigned to each database owner.


### `managed_roles_query`  { #postgres-managed-roles-query }

[managed_roles_query]: #postgres-managed-roles-query

The SQL query to list the name of managed roles.

ldap2pg restricts role deletion and privilege edition to managed roles.
Usualy, this query returns children of a dedicated group like `ldap_roles`.
By default, ldap2pg manages all roles it has access to.

`public` is a special builtin role in Postgres.
If `managed_roles_query` returns `public` role in the list, ldap2pg will manage privileges on `public`.
By default, ldap2pg manages `public` privileges.

The following example tells ldap2pg to manage `public` role, `ldap_roles` and
any members of `ldap_roles`:

``` yaml
postgres:
  managed_roles_query: |
    VALUES
      ('public'),
      ('ldap_roles')

    UNION

    SELECT DISTINCT role.rolname
    FROM pg_roles AS role
    JOIN pg_auth_members AS ms ON ms.member = role.oid
    JOIN pg_roles AS parent
      ON parent.rolname = 'ldap_roles' AND parent.oid = ms.roleid
    ORDER BY 1;
```


### `roles_blacklist_query`  { #postgres-roles-blacklist-query }

[roles_blacklist_query]: #postgres-roles-blacklist-query

The SQL query returning name and glob pattern to blacklist role from management.
ldap2pg won't touch anything on these roles.
Default value is `[postgres, pg_*]`.
ldap2pg blacklist self user.


``` yaml
postgres:
  roles_blacklist_query:
  - postgres
  - "pg_*"
  - "rds_*"
```

!!! warning

    Beware that `*foo` is a YAML reference. You must quote pattern *beginning* with `*`.


### `schemas_query`  { #postgres-schemas-query }

[schemas_query]: #postgres-schemas-query

The SQL query returning the name of managed schemas in a database.
ldap2pg executes this query on each databases returned by `databases_query`,
only if ldap2pg manages privileges.
ldap2pg loops on objects in theses schemas when inspecting GRANTs in the cluster.

``` yaml
postgres:
  schemas_query: |
    SELECT nspname FROM pg_catalog.pg_namespace
```


## PostgreSQL Privileges Section  { #privileges }

[privileges]: #privileges

The `privileges` top level section is a mapping defining privilege profiles,
referenced later in Synchronisation map's [grant rule].
A privilege profile is a list of either a reference to a privilege type in a Postgres ACL or other profile.
A privilege profile may include another profile, recursively.
See [Managing Privileges] for details.

``` yaml
privileges:
  reading:
  - default: global
    type: SELECT
    on: TABLES

  writing:
  - reading
  - default: global
    type: SELECT
    on: TABLES
```

A privilege profile whose name starts with `_` is inactive unless included in an active profile.


### `default` { #privileges-default }

Defines the scope of default privileges.
Can be undefined or either `global` or `schema`.
`global` scope references default privileges for any schemas,
including future schemas.
`schema` scope references default privileges on specific schemas.
Target schema is defined by [grant rule].

``` yaml
privileges:
  reading:
  - default: global
    type: SELECT
    on: TABLES
```


### `type`  { #privileges-type }

Type of privilege as described in [Section 5.7 of PostgreSQL documentation].
e.g. SELECT, REFERENCES, USAGE, etc.

[Section 5.7 of PostgreSQL documentation]: https://www.postgresql.org/docs/current/ddl-priv.html

``` yaml
privileges:
  reading:
  - type: USAGE
    on: SCHEMAS
```


### `on`  { #privileges-on }

Target ACL of privilege type.
e.g. TABLES, SEQUENCES, SCHEMAS, etc.
Note the special cases `ALL TABLES`, `ALL SEQUENCES`, etc.
See [Managing Privileges] documentation for details.

``` yaml
privileges:
  reading:
  - type: SELECT
    on: ALL TABLES
```


## Synchronisation rules  { #rules }

The top level `rules` section is a YAML list.
This is the only mandatory parameter in `ldap2pg.yaml`.
Each item of `rules` is called a *mapping*.
A mapping is a YAML dict with any of `role` or `grant` subsection.
A mapping can optionnaly have a `description` field and a `ldapsearch` section.

``` yaml
rules:
- description: "Define DBA roles"
  ldapsearch:
    base: ...
  roles:
  - name: "{cn}"
    options: LOGIN SUPERUSER
```

The `ldapsearch` subsection is optional.
You can define roles and grants without querying a directory.


### `description`  { #rules-description }

A free string used for logging.
This parameter does not accepts mustache parameter injection.


### `ldapsearch`  { #rules-ldapsearch }

This directive defines LDAP search parameters.
It is named after the ldapsearch CLI utility shipped by OpenLDAP project.
It's behaviour should be mostly the same.

!!! note

    This documentation refers LDAP query as *search*
    while the word query is reserved for SQL query.


`ldapsearch` directives allows and requires LDAP attributes injection in `role` and `grant` rules
using curly braces.
See [Searching directory] for details.

[Searching directory]: ldap.md


#### `base`, `scope` and `filter`  { #ldapsearch-parameters }

These parameters have the same meaning, definition and default as searchbase, scope and filter arguments of ldapsearch CLI utility.

``` yaml
rules:
- ldapsearch:
    base: ou=people,dc=acme,dc=tld
    scope: sub
    filter: >
      (&
         (member=*)
         (cn=group_*)
      )
```


#### `joins`  { #ldapsearch-joins }

Customizes LDAP sub-searches.
The `joins` section is a dictionary with attribute name as key and LDAP search parameters as value.
LDAP search parameters are the same as for top LDAP search.

``` yaml
rules:
- ldapsearch:
    joins:
      member:
        filter: ...
        scope: ...
  role:
  - name: "{member.sAMAccountName}"
```

The search base of sub-search is the value of the referencing attribute,
e.g. each value of `member`.
You can't customize the `base` attribute of sub-search.
Likewise, ldap2pg infers attributes of sub-searches from `role` and `grant` rules.
You can have only a single sub-search per top-level search.
You can't do sub-sub-search.

See [Searching directory] for details.

!!! notice

    Executing a sub-search for each entry of a result set can be very heavy.
    You may optimize the query by using special LDAP search filter like `memberOf`.
    Refer to your LDAP directory administrator and documentation for details.


### `role`  { #rules-role }

[role rule]: #rules-role

Defines a rule to describe one or more roles wanted in the target Postgres cluster.
This includes name, options, config, comment and membership.
Plural form `roles` is valid.
The value can be either a single role rule or a list of role rules.

``` yaml
rules:
- role:
    name: dba
    options: SUPERUSER LOGIN
- roles:
  - name: group0
    options: NOLOGIN
  - name: group1
    options: NOLOGIN
```


#### `comment`  { #role-comment }

Defines the SQL comment of a role.
Default value is `Managed by ldap2pg`.
Accepts LDAP attribute injection.

In case of LDAP attributes injection,
you must take care of how many combination will be generated.
If the template generates a single comment,
ldap2pg will copy the comment for each role generated by the [role rule].
If the template generates multiple comments, ldap2pg associates name and comment.
If there is more or less comments generated than name generated, ldap2pg fails.

The following example defines a static comment shared by all generated roles:

``` yaml
rules:
- roles:
    names:
    - alice
    - bob
    comment: "Static roles from YAML."
```

The following example generates a single comment from LDAP entry distinguised name, copied for all generated roles:

``` yaml
rules:
- ldapsearch:
    ...
  role:
    name: "{cn}"
    comment: "Generated from LDAP entry {dn}."
```

The following example generate a unique comment for each roles generated:

``` yaml
rules:
- ldapsearch:
    ...
  role:
    name: "{member.cn}"
    comment: "Generated from LDAP entry {member}."
```

!!! tip

    If a role is defined multiple times, parents are merged.
    Other fields are kept as declared by the first definition of the role.


#### `name`  { #role-name }

Name of the role wanted in the cluster.
The value can be either a single string or a list of strings.
Plural form `names` is valid.
You can inject LDAP attributes in name using curly braces.
When multiple names are defined, a new role is defined for each name,
each with the same attributes such as `options` and `parents`.
`comment` parameter has a special handling, see [above](#role-comment).

``` yaml
rules:
- roles:
    name: "my-role-name"
```

When injecting LDAP attribute in name,
each value of the LDAP attribute of each LDAP entry will define a new role.
When multiple LDAP attributes are defined in the format,
all combination of attributes are generated.

ldap2pg protects role name with double quotes in the target Postgres cluster.
Capitalization is preserved, spaces are allowed (even if it's a really bad idea).

ldap2pg applies [roles_blacklist_query] on this parameter.


#### `options`  { #role-options }

Defines PostgreSQL role options.
Maybe an SQL-like string or a YAML dictionary.
Valid options are `BYPASSRLS`, `CONNECTION LIMIT`, `LOGIN`, `CREATEDB`, `CREATEROLE`, `INHERIT`, `REPLICATION` and `SUPERUSER`.
Available options varies following the version of the target PostgreSQL cluster and the privilege of ldap2pg user.

``` yaml
- roles:
  - name: my-dba
    options: LOGIN SUPERUSER
  - name: my-group
    options:
      LOGIN: no
      INHERIT: yes
```


#### `config`  { #role-config }

Defines PostgreSQL configuration parameters that will be set for the role.
Must be a YAML dictionary.
Available configuration parameters varies following the version of the target PostgreSQL cluster.
Some parameters requires superuser privileges to be set.
ldap2pg will fails if it does not have privilege to set a config parameter.

``` yaml
- roles:
  - name: my-db-writer
    config:
      log_statement: mod
      log_min_duration_sample: 100
```

Setting `config` to `null` (the default) will disable the feature for the role.
If `config` is a dict, ldap2pg will drop parameter set in cluster but not defined in ldap2pg YAML.
To reset all parameters, set `config` to an empty dict like below.

``` yaml
- roles:
  - name: reset-my-configuration
    config: {}
```

Note that LDAP attributes are not expanded in config values.


#### `parent`  { #role-parent }

Name of a parent role.
A list of names is accepted.
The plural form `parents` is valid too.
Parent role is granted with `GRANT ROLE parent TO role;`.
`parent` parameter accepts LDAP attributes injection using curly braces.
ldap2pg applies [roles_blacklist_query] on this parameter.
Reference parent can be local roles not managed by ldap2pg.

``` yaml
rules:
- role:
    name: myrole
    parent: myparent
```


#### `before_create`  { #role-before-create }

SQL snippet to execute before role creation.
`before_create` accepts LDAP attributes injection using curly braces.
You are responsible to escape attribute with either `.identifier()` or `.string()`.

``` yaml
rules:
- ldapsearch: ...
  role:
    name: "{cn}"
    before_create: "INSERT INTO log VALUES ({cn.string()})"
```


#### `after_create` { #role-after-create }

SQL snippet to execute after role creation.
`after_create` accepts LDAP attributes injection using curly braces.
You are responsible to escape attribute with either `.identifier()` or `.string()`.

``` yaml
rules:
- ldapsearch: ...
  role:
    name: "{sAMAccountName}"
    after_create: "CREATE SCHEMA {sAMAccountName.identifier()} AUTHORIZATION {sAMAccountName.identifier()}"
```


### `grant`  { #rules-grant }

[grant rule]: #rules-grant

Defines a grant of a privilege to a role with corresponding parameters.
Can be a mapping or a list of mapping.
Plural form `grants` is valid too.

``` yaml
rules:
- grant:
    privilege: reader
    databases: __all__
    schema: public
    role: myrole
```


#### `database`  { #grant-database }

Scope the grant to one or more databases.
May be a list of names.
Plural form `databases` is valid.
Special value `__all__` expands to all managed databases as returned by [databases_query].
Defaults to `__all__`.
Grants found in other databases will be revoked.
Accepts LDAP attributes injection using curly braces.

This parameter is ignored for instance-wide privileges (e.g. on LANGUAGE).


#### `privilege`  { #grant-privilege }

Name of a privilege, within the privileges defined in [privileges] YAML section.
May be a list of names.
Plural form `privileges` is valid.
Required, there is not default value.
Accepts LDAP attribute injection using curly braces.


#### `role`  { #grant-role }

Name of the target role of the grant (*granted role* or *grantee*).
Must be listed by [managed_roles_query].
May be a list of names.
Plural form `roles` is valid.
Accepts LDAP attribute injection using curly braces.
ldap2pg applies [roles_blacklist_query] on this parameter.


#### `schema`  { #grant-schema }

Name of a schema, whithin the schemas returned by [schemas_query].
Special value `__all__` means *all managed schemas in the databases*.
May be a list of names.
Plural form `schemas` is valid.
Accepts LDAP attribute injection using curly braces.

This parameter is ignored for privileges on `DATABASE` and other instance-wide or database-wide privileges.


#### `owner`  { #grant-owner }

Name of role to configure default privileges for.
Special value `__auto__` fallbacks to managed roles having `CREATE` privilege on the target schema.
May be a list of names.
Plural form `owners` is valid.
Accepts LDAP attribute injection using curly braces.
