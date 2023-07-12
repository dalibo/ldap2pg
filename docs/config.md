<!-- markdownlint-disable MD033 MD041 MD046 -->

<h1>ldap2pg.yml file reference</h1>

ldap2pg requires a YAML configuration file usually named `ldap2pg.yml` and put
in working directory. Everything can be configured from the YAML file: Postgres
inspect queries, LDAP searches, privileges and synchronization map.

!!! warning

    ldap2pg **requires** a config file where the synchronization map
    is described.


## File Location

ldap2pg searches for configuration file in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set, ldap2pg skip searching the standard
file locations. You can specify `-` to read configuration from standard input.
This is helpful to feed ldap2pg with dynamic configuration.


## File Structure

`ldap2pg.yml` is split in several sections :

- `postgres` : setup Postgres connexion and inspection queries.
- `ldap` : setup LDAP connexion.
- `privileges` : the definition of privileges.
- `sync_map` : the list of LDAP searches and associated mapping to roles and
  grants.

We provide a simple well commented
[ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml),
tested on CI. If you don't know how to begin, it is a good starting point.

!!! note

    If you have trouble finding the right configuration for your needs, feel free to
    [file an issue](https://github.com/dalibo/ldap2pg/issues/new) to get help.


### About YAML

YAML is a super-set of JSON. A JSON document is a valid YAML document. YAML is
a very permissive format where indentation is meaningful. See [this YAML
cheatsheet](https://medium.com/@kenichishibata/yaml-to-json-cheatsheet-c3ac3ef519b8)
for some example.

In `ldap2pg.yaml` file, you will likely use wildcard for glob pattern and curly
brace for LDAP attribute injection. Take care of protecting these characters
with quotes.


## Postgres Parameters

The `postgres` section defines connection parameters and custom SQL queries for
Postgres inspection.

The `postgres` section contains several `*_query` parameters. These parameters
can be either a string containing an SQL query or a YAML list to return a
static list of values, skipping execution of a query on PostgreSQL cluster.


### `dsn`  { #postgres-dsn }

Specifies a PostgreSQL connexion URI.

``` yaml
postgres:
  dsn: postgres://user@%2Fvar%2Frun%2Fpostgresql:port/
```

!!! warning

    ldap2pg refuses to read a password from a group readable or world
    readable `ldap2pg.yml`.


### `databases_query`  { #postgres-databases-query }

[databases_query]: #postgres-databases-query

The SQL query to list databases in the cluster. This defaults to all databases
connectable, thus including `template1`. ldap2pg uses this list to reassign
objects before dropping a role and managed privileges of databases and other
objects in each databases.

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

Name of the role accepting ownership of database of dropped role. Defaults to
role used by ldap2pg to synchronize cluster.

Before dropping a role, ldap2pg reassign objects and purge ACL. ldap2pg starts
by reassigning database owner by the targetted user. The new owner of the
database is the *fallback owner*. Other objects are reassigned to each database
owner.


### `managed_roles_query`  { #postgres-managed-roles-query }

[managed_roles_query]: #postgres-managed-roles-query

The SQL query to list the name of managed roles. ldap2pg restricts role
deletion and privilege edition to managed roles. Usualy, this query returns
children of a dedicated group like `ldap_roles`. By default, ldap2pg manages
all roles found.

`public` is a special builtin role in Postgres. If `managed_roles_query`
returns `public` role in the list, ldap2pg will manage privileges on `public`.
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


### `owners_query`  { #postgres-owners-query }

The SQL query to list the names of object owners. ldap2pg execute this query
*once* in the cluster, after ldap2pg has created all roles, before granting and
revoking privileges. You need this query only if you manage default privileges
with ldap2pg.

``` yaml
postgres:
  owners_query: |
    SELECT role.rolname
    FROM pg_catalog.pg_roles AS role
    WHERE role.rolsuper IS TRUE;
```

You can declare per-schema owners with [schemas_query]. See [Managing
Privileges] for details.

[Managing privileges]: privileges.md


### `roles_blacklist_query`  { #postgres-roles-blacklist-query }

[roles_blacklist_query]: #postgres-roles-blacklist-query

The SQL query returning name and glob pattern to blacklist role from
management. ldap2pg won't touch anything on these roles. Default value is
`[postgres, pg_*]`.

``` yaml
postgres:
  roles_blacklist_query:
  - postgres
  - "pg_*"
  - "rds_*"
```

!!! warning

    Beware that `*foo` is a YAML reference. You must quote pattern *beginning* with
    `*`.


### `roles_query`  { #postgres-roles-query }

The SQL query returning all roles, their options and their members. It's not
very useful to customize this. Prefer to configure `roles_blacklist_query` and
`managed_roles_query` to confine synchronization to a subset of roles.

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


### `schemas_query`  { #postgres-schemas-query }

[schemas_query]: #postgres-schemas-query

The SQL query returning the name of schemas in a database. ldap2pg executes
this query on each databases returned by `databases_query`, only if ldap2pg
manages privileges. ldap2pg loops on objects in theses schemas when inspecting
GRANTs in the cluster.

``` yaml
postgres:
  schemas_query: |
    SELECT nspname FROM pg_catalog.pg_namespace
```

`schema_query` can return a second column as an array of string. This column
defines the name of roles owning objects in this schema. See [Managing
Privileges] for details.


## LDAP Directory Connection  { #ldap }

The LDAP section is fairly simple. The top-level `ldap` section is meant only
to gather LDAP connexion informations.

The `ldap` section defines libldap parameters.

``` yaml
ldap:
  uri: ldap://ldap2pg.local:389
  binddn: cn=admin,dc=ldap2pg,dc=local
  # For SASL
  sasl_mech: DIGEST-MD5
  user: saslusername
  password: SECRET
```

Actually, it's better to configure ldap connexion through `ldaprc` and regular
libldap environment variables than in YAML. See ldap.conf(5) for details.

The best practice is to configure ldapsearch and then ldap2pg will be happy
like any other libldap tool. If not, please [open an issue]. This allows you to
share `ldap2pg.yml` file between different environments.

[open an issue]: https://github.com/dalibo/ldap2pg/issues/new


## PostgreSQL Privileges Section  { #privileges }

[privileges]: #privileges

The `privileges` top level section is a mapping defining high-level privileges
in Postgres cluster, referenced later in Synchronisation map [grant rule].

An entry in `privileges` is either a list of other privileges (also known as
group of privileges) or a definition of a custom privilege. A group of
privileges can include another group of privileges.

A privilege whose name starts with `_` is inactive. An active privilege grant
found in the cluster and not granted in the YAML is implicitly revoked. An
inactive privilege is still available for inclusion in a privilege group. Every
privileges included in an active group is activated, regardless of the `_`
prefix. This allows ldap2pg to ship [well-known privileges](wellknown.md) ready
to be included in your custom group of privileges.

``` yaml
privileges:
  my-custom-privilege-group:
  - __select_on_tables__
  - __connect__
  - custompriv

  my-custom-privilege:
    type: datacl
    inspect: SELECT ...
    grant: GRANT ...
    revoke: REVOKE ...
```

Writing a custom privilege is hard. Before writing one, ensure that it's not
already shipped in ldap2pg [well-known privileges](wellknown.md). Please open
an issue if you miss a builtin privilege. Also, please share your custom
privilege. This will increase the quality of privilege handling in your
installation and in ldap2pg project.


### `type`  { #privileges-type }

Privilege can be of different kind. The type of privilege influences whether
ldap2pg should loops on databases, schemas or owners roles. This changes also
the parameters required to define a grant.

``` yaml
privileges:
  my-custom-privilege:
    type: datacl
```

See [Defining Custom Privileges] for possible values and their meaning.

[defining custom privileges]: privileges.md#defining-custom-privileges


### `inspect`  { #privileges-inspect }

The SQL query to inspect grants of this privilege in the cluster. The signature
of tuples returned by this query varies after privilege type. This query may be
executed once for global objects or per database, depending on privilege type.

``` yaml
privileges:
  my-custom-privilege:
    inspect: |
      SELECT grantee FROM ...
```

This is the trickiest query to write when synchronizing privileges. See
[Privilege documentation](privileges.md) for details.


### `grant`  { #privileges-grant }

SQL query to grant a privilege to a role. Some parameters are injected in this
query using mustache substitution like `{role}`. Parameters depends on
privilege type. For example, a defacl privilege must accept an `{owner}`
parameter.

This option must not be confused with [grant rule] in synchronisation map.

``` yaml
privileges:
  my-custom-privilege:
    grant: GRANT SELECT ON ALL TABLES IN SCHEMA {schema} TO {role};
```

See [Privilege documentation](privileges.md) for details.


### `revoke`  { #privileges-revoke }

Just like `grant` ; the SQL query to revoke a privilege from a role. Parameters
are substituted with mustache syntax like `{role}` and depends on privilege
type. You must reuse the same parameters as in `grant` query.

``` yaml
privileges:
  my-custom-privilege:
    revoke: REVOKE SELECT ON ALL TABLES IN SCHEMA {schema} FROM {role};
```

See [Privilege documentation](privileges.md) for details.


## Synchronisation map  { #sync-map }

The top level `sync_map` section is a YAML list. This is the only mandatory
parameter in `ldap2pg.yaml`. Each item of `sync_map` is called a *mapping*. A
mapping is a YAML dict with a `description` field and any of `ldapsearch`,
`role` and `grant` subsection.

``` yaml
sync_map:
- description: "Define DBA roles"
  ldapsearch:
    base: ...
  roles:
  - name: "{cn}"
    options: LOGIN SUPERUSER
```

The `ldapsearch` subsection is optional. You can define roles and grants
without querying a directory.


### `description`  { #sync-map-description }

A free string used for logging. This parameter does not accepts mustache
parameter injection.


### `ldapsearch`  { #sync-map-ldapsearch }

This directive defines LDAP search parameters. Not to be confused with
top-level `ldap` section defining LDAP connexion parameters. It is named after
the ldapsearch CLI utility shipped by OpenLDAP project. It's behaviour should
be mostly the same.

!!! note

    This documentation refers LDAP query as *search* while the word query
    is reserved for SQL query.


`ldapsearch` directives allows and requires LDAP attributes injection in `role`
and `grant` rules using curly braces. See [Searching directory] for
details.

[Searching directory]: ldap.md


#### `allow_missing_attributes`  { #ldapsearch-allow-missing-attributes }

Lists the names of LDAP attributes that LDAP server may not return. This
parameters configures a ldap2pg behavior to prevent typographic error in
configuration. LDAP protocols allows to query arbiratry attributes, event
undefined ones. If configuration has a typo, no error will be returned. By
default, ldap2pg consider a missing attributes as a typo, except if it's listed
in `allow_missing_attributes`.

If `member` attribute is searched, it will be added to the list. This is the
default value.

The following example accepts that a LDAP entry miss `sAMAccountName`. The
entry missing `sAMAccountName` wont generate a role.

``` yaml
sync_map:
- ldapsearch:
    base: ...
    allow_missing_attributes: [member, sAMAccountName]
  roles:
  - name: "{sAMAccountName}"
    members: "{member.cn}"
```


#### `base`, `scope` and `filter`  { #ldapsearch-parameters }

These parameters have the same meaning, definition and default as searchbase,
scope and filter arguments of ldapsearch CLI utility.

``` yaml
sync_map:
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

Customizes LDAP sub-searches. The `joins` section is a dictionary with
attribute name as key and LDAP search parameters as value. LDAP search
parameters are the same as for top LDAP search.

``` yaml
sync_map:
- ldapsearch:
    joins:
      member:
        filter: ...
        scope: ...
  role:
  - name: "{member.sAMAccountName}"
```

The search base of sub-search is the value of the referencing attribute, e.g.
each value of `member`. You can't customize the `base` attribute of sub-search.
Likewise, ldap2pg infers attributes of sub-searches from `role` and `grant`
rules.

See [Searching directory] for details.

!!! notice

    Executing a sub-search for each entry of a result set can be very heavy. You
    may optimize the query by using special LDAP search filter like `memberOf`.
    Refer to your LDAP directory administrator and documentation for details.


#### `on_unexpected_dn`  { #ldapsearch-on-unexpected-dn }

Sometimes, an attribute references another entry in LDAP rather than specifying
a value. This mixed types attributes are hard to handle and must be avoided.

The `on_unexpected_dn` parameter allows you to tell ldap2pg how to behave it
this case. The default is to fail. You can choose to either `warn` or silently
`ignore` these values.

``` yaml
sync_map:
- ldapsearch:
    on_unexpected_dn: warn  # fail | warn | ignore
```


### `role`  { #sync-map-role }

[role rule]: #sync-map-role

Defines a rule to define one or more roles wanted in the target Postgres
cluster. This includes name, options, comment and membership. Plural form
`roles` is valid. The value can be either a single role rule or a list of role
rules.

``` yaml
sync_map:
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

Defines the SQL comment of a role. Default value is `Managed by ldap2pg`.
Accepts LDAP attribute injection.

In case of LDAP attributes injection, you must take care of how many
combination will be generated. If the template generates a single comment,
ldap2pg will copy the comment for each role generated by the [role rule]. If
the template generates multiple comments, ldap2pg associates name and comment.
If there is more or less comments generated than name generated, ldap2pg fails.

The following example defines a static comment shared by all generated roles:

``` yaml
sync_map:
- roles:
    names:
    - alice
    - bob
    comment: "Static roles from YAML."
```

The following example generate a single comment from LDAP entry distinguised
name, copied for all generated roles:

``` yaml
sync_map:
- ldapsearch:
    ...
  role:
    name: "{cn}"
    comment: "Generated from LDAP entry {dn}."
```

The following example generate a unique comment for each roles generated:

``` yaml
sync_map:
- ldapsearch:
    ...
  role:
    name: "{member.cn}"
    comment: "Generated from LDAP entry {member}."
```


#### `member`  { #role-member }

Name of a child role. A list of names is accepted. The plural form `members` is
valid. Role is granted to children with `GRANT ROLE role TO member`. `member`
parameter accepts LDAP attributes injection using curly braces. ldap2pg applies
[roles_blacklist_query] on this parameter.

``` yaml
sync_map
- role:
    name: myrole
    member: mychild
```


#### `name`  { #role-name }

Name of the role wanted in the cluster. The value can be either a single string
or a list of strings. Plural form `names` is valid. You can inject LDAP
attributes in name using curly braces. When multiple names are defined, a new
role is defined for each name, each with the same attributes such as `options`
and `members`. `comment` parameter has a special handling, see
[above](#role-comment).

``` yaml
sync_map:
- roles:
    name: "my-role-name"
```

When injecting LDAP attribute in name, each value of the LDAP attribute of each
LDAP entry will define a new role. When multiple LDAP attributes are defined in
the format, all combination of attributes are generated.

ldap2pg protects role name with double quotes in the target Postgres cluster.
Capitalization is preserved, spaces are allowed (even if it's a really bad
idea).

ldap2pg applies [roles_blacklist_query] on this parameter.


#### `name_match`  { #role-name-match }

Defines a condition on name as a glob pattern. If a name does not match this
pattern, the role is skipped from creation.

``` yaml
sync_map:
- role:
    name: "{cn}"
    name_match: "external_*"
```


#### `options`  { #role-options }

Defines PostgreSQL role options. Maybe an SQL-like string or a YAML dictionary.
Valid options are `BYPASSRLS`, `LOGIN`, `CREATEDB`, `CREATEROLE`, `INHERIT`,
`REPLICATION` and `SUPERUSER`. Available options varies following the version
of the target PostgreSQL cluster.

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
Must be a YAML dictionary. Available configuration parameters varies following
the version of the target PostgreSQL cluster.

``` yaml
- roles:
  - name: my-db-writer
    config:
      log_statement: mod
      log_min_duration_sample: 100
```

If no block is specified, then ldap2pg will ignore any existing configuration
parameters. Otherwise, any existing configuration parameters that do not match
those listed in the `config` block will be reset to their default values.

You can specify that all configuration parameters are reset to default with
an empty config block, e.g.

``` yaml
- roles:
  - name: reset-my-configuration
    config: {}
```

Note that LDAP attributes are not expanded in config values.

#### `parent`  { #role-parent }

Name of a parent role. A list of names is accepted. The plural form `parents`
is valid too. Parent role is granted with `GRANT ROLE parent TO role;`.
`parent` parameter accepts LDAP attributes injection using curly braces.
ldap2pg applies [roles_blacklist_query] on this parameter.

``` yaml
sync_map:
- role:
    name: myrole
    parent: myparent
```


### `grant`  { #sync-map-grant }

[grant rule]: #sync-map-grant

Defines a grant of a privilege to a role with corresponding parameters.

``` yaml
sync_map:
- grant:
    privilege: reader
    databases: __all__
    schema: public
    role: myrole
```


#### `database`  { #grant-database }

Name of a database, within the list of database names returned by
[databases_query]. May be a list of names. Plural form `databases` is valid.
Default value is special name `__all__` which mean all databases as returned by
[databases_query]. Defines the database where the privilege must be granted.
Grants found in other databases will be revoked. Accepts LDAP attributes
injection using curly braces.

This parameter is valid for all types of privileges.


#### `privilege`  { #grant-privilege }

Name of a privilege, within the privileges defined in [privileges] YAML
section. May be a list of names. Plural form `privileges` is valid. Required,
there is not default value. Accepts LDAP attribute injection using curly
braces.


#### `role`  { #grant-role }

Name of the target role of the grant (*granted role*). Must be returned in the
result of [managed_roles_query]. May be a list of names. Plural form `roles` is
valid. Accepts LDAP attribute injection using curly braces. ldap2pg applies
[roles_blacklist_query] on this parameter.


#### `schema`  { #grant-schema }

Name of a schema, whithin the schemas returned by [schemas_query]. Special
value `__all__` means *all managed schemas in the databases* returned by
[schemas_query]. May be a list of names. Plural form `schemas` is valid.
Accepts LDAP attribute injection using curly braces.

This parameter is ignored for privileges of type `globaldefacl` and `datacl`.
