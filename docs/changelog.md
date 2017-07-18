<!--*- markdown -*-->

<h1>Changelog</h1>

Here is a highlight of changes in each versions. If you need further details,
just
follow
[merged Pull request pages](https://github.com/dalibo/ldap2pg/pulls?utf8=%E2%9C%93&q=is%3Apr%20is%3Amerged).

# ldap2pg 2.0

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
