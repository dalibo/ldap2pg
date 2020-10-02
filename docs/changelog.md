<!-- markdownlint-disable MD033 MD041 -->

<h1>Changelog</h1>

Here is a highlight of changes in each versions. If you need further details,
follow [merged Pull request
pages](https://github.com/dalibo/ldap2pg/pulls?utf8=%E2%9C%93&q=is%3Apr%20is%3Amerged).


# ldap2pg 5.5

- Permit joins where all the referenced objects are filtered out. The join
  name must be added to the list of attributes that may be missing in the
  result.
- Fail when attribute is misspelled. You must explicitly list attributes that
  may be missing in the result. By default, ldap2pg accepts missing `member`
  and considers it an empty list rather than a misspelled attribute.
- Rewrite string generation from LDAP attribute to fix corner cases and
  inconsistency.
- Fail when sub-querying on bad DN.
- CentOS 8 support.
- PostgreSQL 13 support.
- Fix join order.


# ldap2pg 5.4

- Fix grant to capitalized role.
- Fix rename of members.
- Log role after their originated LDAP query.
- Add `description:` to mapping for logging.


# ldap2pg 5.3

- Fix join when multiple entries are returned.
- Fix using multiple attributes from joined entries.
- Fix comment error with generated comment.
- Fails if configuration file is not found.
- Refuse empty configuration file.
- Refuse undefined `sync_map`.
- Update [sample
  ldap2pg.yml](https://github.com/dalibo/ldap2pg/bloc/master/ldap2pg.yml) for
  readability and general use.


# ldap2pg 5.2

**Attention!** This release has some behaviour changes. Some silenced errors
are now raised when encountered. Please test on staging environment before
deploying on production.

- Fix ignored LDAP entries after unexpected DN.
- Fix traceback when inspecting grants.
- Fix role comment overridden on alter role.
- Fix default configuration filename in ~/.config.
- Refuse to mix static rules and ldap query.
- Accepts an SQL query to list ignored roles. `postgres:blacklist` is renamed
  `postgres:roles_blacklist_query`. ldap2pg ensure backward compatibility.
- Apply roles blacklist to LDAP results.
- Generate unique comment per role instead of shared comment per rule.
- Move `on_unexpected_dn` to `ldap` query.


# ldap2pg 5.1

**Beware** when upgrading : **ldap2pg will rename roles having uppercase letter
in their name!** These roles will be renamed from lowercase to original case.
Run `ldap2pg --dry` before and check for renames.

- ldap2pg now respect case for role names. Thanks to [Sergejs
  Zuromskis](https://github.com/zurikus) for the report.
- Postgres 12 support validated.
- Fix void attributes raising *Missing attribute error*.
- Docker image now ensure ldap2pg is pinned to the desired version.
- Moved to new homepage : [labs.dalibo.com/ldap2pg](https://labs.dalibo.com/ldap2pg).


# ldap2pg 5.0

- Fix default ldap settings overriding ldaprc values.
- Allow joining LDAP entries based on DN attributes, e.g. to support role name
  synchronization using the Active Directory (AD) attributes `sAMAccountName`
  or `userPrincipalName`.
- Let user choose psycopg2 distribution. Affects only pip.
- Support GSSAPI authentification for Kerberos. Thanks @djkube for testing.


# ldap2pg 4.18

- Fix ref discarding.
- Ship official docker image: dalibo/ldap2pg.
- Parse LDAP settings from YAML too.


# ldap2pg 4.17

- Fix broken `__usage_on_types__`. Replaced by `__default_usage_on_types__`.
- Gently raise connection errors.
- Warn on possible typo in config key.


# ldap2pg 4.16

- Allow to customize comment on role creation.
- Fix decoding Postgres error with utf-8 chars.
- Include foreign tables in inspect ON ALL TABLES grants.


# ldap2pg 4.15

- Add Amazon RDS admin roles in default blacklist.
- Skip `pg_temp_*` and `pg_toast_temp_*` schemas when inspecting grants.
- Fix schema naïve privilege inspection.
- Fix newly created roles excluded from privilege inspection.
- Time LDAP searches, Postgres inspection and Postgres synchronization. Time
  delta are shown in debug messages.
- Trace maximum memory used in debug message.
- Reduce memory usage of grants and roles.


# ldap2pg 4.14

- Allow to exclude public from managed roles. When scoping ldap2pg to a subset
  of roles, ldap2pg was including the public role, always. Now you can include
  or exclude public by using `managed_roles_query` parameter. If you customized
  `managed_roles_query` you **must update ldap2pg.yml** to include `public` to
  keep the same behaviour. See [Synchronize a subset of
  roles](postgres.md#synchronize-a-subset-of-roles) documentation section.


# ldap2pg 4.13

- Allow to configure behaviour on unexpected DN. Current behaviour are `ignore`,
  `warn` and `fail`. If a LDAP attribute has references different objectClass,
  accessing a RDN triggers an error. The `on_unexpected_dn` configuration key
  allows to configure this behaviour.
- LDAPREFERRALS is now disabled by default, just like ldapsearch and other
  openldap tools. You must explicitly enable REFERRALS with `LDAPREFERRALS=yes`
  env var, or `REFERRALS yes` ldap.conf(1) parameter.


# ldap2pg 4.12

- Fix Bad search filter when using multiline YAML string.
- Fix support for Postgres 9.3.


# ldap2pg 4.11

- Use PyYAML safe loading.
- Don't log `-D` switch for anonymous `ldapsearch`.
- Refuse useless LDAP queries without attributes.
- Manage binary decoding error.
- Fix gathering of LDAP attributes on Python 2.


# ldap2pg 4.10

- Fine grained logging setup.
- Unify `roles` and `role_attribute` with string formatting.


# ldap2pg 4.9

- Fix mix of parents in same role rule
- Renamed `acl` to `privilege` in configuration. See documentation for details.
- Run as non-superuser, in a degraded mode. See Cookbook for details.


# ldap2pg 4.8

- Fix traceback on unknown schema.
- Check YAML gotchas.
- Allow to define role option once even when defining roles twice.
- pyldap has been merged in python-ldap. Dropping pyldap.


If you use pyldap to run ldap2pg on Python3, please either :

- uninstall pyldap and switch to python-ldap 3.0.0. Do this in two steps : see
  https://github.com/pyldap/pyldap/issues/148 for details.
- switch to Python2 and use python-ldap.
- keep running ldap2pg 4.7.


# ldap2pg 4.7

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
- Fix traceback on nonexistent database in ACL.
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
- SASL authentication support.
- Read configuration from stdin.


# ldap2pg 1.0

- Bootstrap project
- Automatic unit and functional tests.
- Read configuration from CLI arguments, env vars and YAML.
- Manage Postgres roles, role options and role members.
- Creates roles from LDAP entries or from static values in YAML.
- Verbose mode with Postgres and LDAP queries logged.

<!-- Local Variables: -->
<!-- ispell-dictionary: "american" -->
<!-- End: -->
