<!--*- markdown -*-->

<h1>Changelog</h1>

Here is a highlight of changes in each versions. If you need further details,
follow [merged Pull request
pages](https://github.com/dalibo/ldap2pg/pulls?utf8=%E2%9C%93&q=is%3Apr%20is%3Amerged).


# ldap2pg 4.7 (unreleased)

- Fix `__usage_on_types__` regranted for each owner.
- Fix `ALTER DEFAULT PRIVILEGES` on blacklisted roles.
- Warn about undetermined `ALTER DEFAULT PRIVILEGES`.
- Sort GRANT/REVOKE by dbname and role first.
- Reuse existing role. Drop roles only from `managed_roles_query`.
- Commit transaction when changing database. This increase performances a lot.


# ldap2pg 4.6

- Allow to inspect owners per schema.
- Use configured database instead of hardcoded `postgres`.
- Increase arbitrary database limit to 256.
- Accept list for `grant:databases` and `grant:schemas`.


# ldap2pg 4.5

- Lint log level and messages.
- Deduplicate LDAP auto-attributes.
- Add `parents_attribute` to fetch parent from LDAP entry.
- Comment roles with `Managed by ldap2pg.`.


# ldap2pg 4.4

- Fix uninitialized ldap parameters.
- Fix `__all_on_schemas__` group including a `sequences` ACL.
- Fix user drop with Postgres 9.4.
- Fix traceback on unknown parent.
- Fix traceback on unknown `PGHOST`.
- Add `*_on_tables__` ACL for all privileges on table.
- Allow to customize managed databases with `postgres:databases_query`.
- Allow pure static configuration (aka ldap2pg without LDAP).
- Don't revoke ACL granted to roles not in `roles_query`.
- Don't revoke ACL granted on schema not in `schemas_query`.


# ldap2pg 4.3.1

- Fix all procs ACL inspection.


# ldap2pg 4.3

- Fix case sensitivity in LDAP query. Thanks @dirks for report and tests.
- Allow to customize owners for `ALTER DEFAULT PRIVILEGES` with
  `postgres:owners_query`.
- Don't execlude `pg_catalog` from `__all__` schema group.
- Allow to customize schema introspection with `postgres:schema_query`.


# ldap2pg 4.2

- Support Postgres 9.4 and lower.
- Manage ACL on views.
- Autogenerate LDAP search attributes from mappings.
- Fix case sensitivity of `*_attribute`.


# ldap2pg 4.1

- Merge role memberships when inspected twice.
- Manage `ALTER DEFAULT PRIVILEGES` on global schema.


# ldap2pg 4.0

- **Deprecation**: use `acls:` rather than `acl_dict` and `acl_groups`.
- **Deprecation**: `sync_map` should be a list.
- **Deprecation**: schema `__all__` should be used instead of `__all__`.
- Fix various tracebacks with errors in configuration or SQL queries.
- Manage grants to `public` role.
- Provide new [well known ACL](wellknown.md) for, `__temporary__`,
  `__create_on_schema__`.
- Provide `__all_on_tables__`, `__all_on_schemas__` and `__all_on_sequences__`
  well known ACL groups.


# ldap2pg 3.4

- Fix unicode error on logging SQL query.
- Fix traceback on inexistant database in ACL.
- Fix various configuration loading errors.
- Fix Distinguished Name case sensitivity.
- Provide [well known ACLs](wellknown.md).
- Merge `acl_dict` and `acl_groups` in `acls`.
- Manage `ALTER DEFAULT PRIVILEGES`.
- Support psycopg2 2.0.
- Support Python 2.6.
- Tested on CentOS 6.
- Show detailed version informations.
- Show YAML parsing error.
- Avoid reading ldaprc twice.
- Quote role name in SQL queries.
- Documentation and sample update.


# ldap2pg 3.3

- Fix unicode management in Python3.
- Check for name or name_attribute in role rule.
- Avoid inspecting schema if only synchronizing roles.


# ldap2pg 3.2

- Manage unicode in role name.
- Tested on Postgres 10.


# ldap2pg 3.1

- Fix unhandled exception when attribute does not exists in LDAP.
- Use LDAP standard default filter `(objectClass=*)`.
- Add auth CLI arguments to logged ldapsearch commands.
- Change *Empty mapping* error to a warning.


# ldap2pg 3.0

- Breakage: Use Python `{}` format string for ACL queries instead of named
  printf style.
- Support old setuptools.
- Fix undefined LDAP password traceback.
- Fix case sensitivity in grant rule.
- ACL inspect query should now return a new column indicating partial grant.
- Allow to customize query to inspect roles in cluster.
- Add check mode: exits with 1 if changes. Juste like diff.
- Add `--quiet` option.
- Add `__all__` schema wildcard for looping all schema in databases.
- Add ACL group to ease managing complex ACL setup.
- Add Cookbook in documentation.


# ldap2pg 2.0

- Adopt new logo.
- Inspect, grant and revoke custom ACLs.
- Reassign objects on role delete.
- Manage several databases.
- Move to libldap through [pyldap](https://github.com/pyldap/pyldap).
- Accept standard libldap `LDAP*` env vars.
- *Deprecation*: `LDAP_*` envvars are deprecated in favor of libldap2 regular
  envvars.
- Read ldaprc files.
- SSL/TLS support.
- SASL authentification support.
- Read configuration from stdin.


# ldap2pg 1.0

- Bootstrap project
- Automatic unit and functionnal tests.
- Read configuration from CLI arguments, env vars and YAML.
- Manage Postgres roles, role options and role members.
- Creates roles from LDAP entries or from static values in YAML.
- Verbose mode with Postgres and LDAP queries logged.
