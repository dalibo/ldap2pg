<h1>Cookbook</h1>

Here in this cookbook, you'll find some recipes for various use case of
`ldap2pg`.

If you struggle to find a way to setup `ldap2pg` for your needs, please [file an
issue](https://github.com/dalibo/ldap2pg/issues/new) so that we can update
*Cookbook* with new recipes ! Your contribution is welcome!


# Configure Postgres with LDAP ?

`ldap2pg` does **NOT** configure PostgreSQL for you. You should carefully read
[PostgreSQL
documentation](https://www.postgresql.org/docs/current/static/auth-methods.html#auth-ldap)
about LDAP authentication for this point. Having PostgreSQL properly configured
**before** writing `ldap2pg.yml` is a good start. Here is the steps to setup
PostgreSQL with LDAP in the right order:

- Write the LDAP query and test it with `ldapsearch(1)`. This way, you can also
  check how you connect to your LDAP directory.
- In PostgreSQL cluster, **manually** create a single role having its password
  in LDAP directory.
- Edit `pg_hba.conf` following [PostgreSQL
  documentation](https://www.postgresql.org/docs/current/static/auth-methods.html#AUTH-LDAP)
  until you can effectively login with the single role and the password from
  LDAP.
- Write a simple `ldap2pg.yml` with only one LDAP query just to setup `ldap2pg`
  connection parameters for PostgreSQL and LDAP connection. `ldap2pg` always run
  in dry mode by default, so you can safely loop `ldap2pg` execution until you
  get it right.
- Then, complete `ldap2pg.yml` to fit your needs following [ldap2pg
  documentation](config.md). Run `ldap2pg` for real and check that ldap2pg
  maintain your single test role, and that you can still connect to the cluster
  with it.
- Finally, you must decide when and how you want to trigger synchronization: a
  regular cron tab ? An ansible task ? Manually ? Other ? Ensure `ldap2pg`
  execution is frequent, on purpose and notified !


# Configure Postgres Connection

The simplest case is to save the connection settings in `ldap2pg.yml`, section
`postgres`:

``` yaml
postgres:
  dsn: postgres://user:password@host:port/
```

`ldap2pg` checks for file mode and refuse to read password in world readable
files. Ensure it is not world readable by setting a proper file mode:

``` console
$ chmod 0600 ldap2pg.yml
```

`ldap2pg` will warn about *Empty synchronization map* and ends with *Comparison
complete*. `ldap2pg` suggests to drop everything. Go on and write the
synchronization map to tell `ldap2pg` the required roles for the cluster.


# Query LDAP

The first step is to query your LDAP server with `ldapsearch`, the CLI tool from
OpenLDAP. Like this:

``` console
$ ldapsearch -H ldaps://ldap.ldap2pg.docker -U testsasl -W -b dc=ldap,dc=ldap2pg,dc=docker
Enter LDAP Password:
SASL/DIGEST-MD5 authentication started
SASL username: testsasl
SASL SSF: 128
SASL data security layer installed.
# extended LDIF
#
# LDAPv3
...
# search result
search: 4
result: 0 Success

# numResponses: 16
# numEntries: 15
$
```

Now save the settings in `ldap2pg.yml`:

``` yaml
ldap:
  uri: ldaps://ldap.ldap2pg.docker
  user: testsasl
  password: "*secret*"
```

Next, update your `ldapsearch` to properly match role entries in LDAP server:

``` console
$ ldapsearch -H ldaps://ldap.ldap2pg.docker -U testsasl -W -b cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker '' member
...
# dba, groups, ldap.ldap2pg.docker
dn: cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker
member: cn=Alan,ou=people,dc=ldap,dc=ldap2pg,dc=docker
member: cn=albert,ou=people,dc=ldap,dc=ldap2pg,dc=docker
member: cn=ALICE,ou=people,dc=ldap,dc=ldap2pg,dc=docker

# search result
search: 4
result: 0 Success

...
$
```

Now translate the query in `ldap2pg.yml` and associate a role mapping to produce
roles from each values of each entries returned by the LDAP search:

``` yaml
- ldap:
    base: cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker
    attributes: member
  role:
    name_attribute: member.cn
    options: LOGIN SUPERUSER
```

Test it:

``` console
$ ldap2pg
...
Querying LDAP cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker...
Would create alan.
Would create albert.
Would update options of alice.
...
Comparison complete.
$
```

Read further on how to control role creation from LDAP entry in
[Configuration](config.md). Once you're satisfied with the comparison output, go
real with `--real`.


# Using LDAP High-Availability

`ldap2pg` supports LDAP HA out of the box just like any openldap client. Use a
space separated list of URI to tells all servers.

``` console
$ LDAPURI="ldaps://ldap1 ldaps://ldap2" ldap2pg
```

See [`ldap.conf(5)`](https://www.openldap.org/software/man.cgi?query=ldap.conf)
for further details.


# Don't Drop Role not in Directory

Usualy, you have roles in the cluster not defined in LDAP directory. At least
`postgres` superuser. You may have other roles that ldap2pg will treat as
spurious roles and would simply drop, because these are not in the directory.

You can define statically the role as if it were in the directory. This makes
sense especially when you require this role to synchronize the others.

``` yaml
sync_map:
- role:
    name: ldap_users
    options: NOLOGIN
- ldap:
    base: ...
  role:
    name_attribute: ...
    parent: ldap_users
```

Another solution is to blacklist the role. `ldap2pg` will always consider this
role as missing in the cluster.

``` yaml
postgres:
  blacklist: [postgres, pg_*, ldap_users]
```


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


# Read-Only / Read-Write / DDL groups

A good pattern to manage ACLs is to have a group with read-only grants, a group
with read-write grants and finally a group with DDL grants. First, declare the
sets of ACL you want to grant to each groups, then apply them to corresponding
roles.


``` yaml
acls:
  ro:
  - __usage_on_schemas__
  - __select_on_tables__
  
  rw:
  - ro
  - __all_on_tables__
  
  ddl:
  - rw
  - __create_on_schemas__

sync_map:
- roles:
  - names: [owners, readers, writers]
    options: NOLOGIN
- grant:
  - acl: ro
    database: mydb
    schema: __all__
    role: readers
  - acl: rw
    database: mydb
    schema: __all__
    role: writers
  - acl: ddl
    database: mydb
    schema: __all__
    role: owners
```


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
