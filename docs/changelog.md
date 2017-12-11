<!--*- markdown -*-->

<h1>Changelog</h1>

Here is a highlight of changes in each versions. If you need further details,
just
follow
[merged Pull request pages](https://github.com/dalibo/ldap2pg/pulls?utf8=%E2%9C%93&q=is%3Apr%20is%3Amerged).


# ldap2pg 3.4 (unreleased)

- Support psycopg2 2.0.
- Support Python 2.6.
- Tested on CentOS 6.
- Fix unicode error on logging SQL query.
- Fix traceback on inexistant database in ACL.
- Avoid reading ldaprc twice.
- Show detailed version informations.
- Show YAML parsing error.
- Fix various configuration loading errors.
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
