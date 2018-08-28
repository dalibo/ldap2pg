<h1>Querying Directory with LDAP</h1>

`ldap2pg` searches for LDAP query in `sync_map` items in the `ldap` entry.

A `ldap` entry contains `base`, `scope` and `filter`. The meaning of `base`,
`scope` and `filter` is strictly the same as in `ldapsearch`. `base` must be
defined. `filter` defaults to `(objectClass=*)`, just like `ldapsearch(1)`.
`scope` defaults to `sub`, likewise. To inject an attribute in a property, use
`{attributename}` format. `ldap2pg` determines which attributes to query
depending on values in `grant` and `roles` rules.

!!! tip

    `ldap2pg` logs LDAP queries as `ldapsearch` commands. You can debug a failing
    query by copy-pasting the command in your shell and update parameters. Once you
    are okay, report back the right parameters.

A LDAP search is **not** mandatory. `ldap2pg` can create roles defined
statically from YAML.

Each LDAP search is done once and only once. There is neither loop nor
deduplication of LDAP searches.


## Mapping LDAP attributes to Postgres

In `role` and `grant` rules, you can inject LDAP attributes using curly brace
formatting. E.g. a role name `{uid}` will generate a role for each `uid` value
in LDAP object.

`ldap2pg` generate a `role` or `grant` for each value of the attribute of each
entries returned by the directory. If there is multiple attributes in the format
string, a product of all combination is generated.

If the attribute is of type Distinguished Name (DN), you can refer to the first
Relative Distinguished Name (RDN) of the value with a dot, like this:
`<attrname>.<rdn>`. For example, if a LDAP entry has `member` attribute with
value `cn=toto,ou=people,dc=ldap,dc=acme,dc=fr`, `name: '{member.cn}''` will
generate `toto`.

You can inject attributes in `role:names`, `role:parents`, `role:members` and
`grant:role`.


## Managing heterogeneous DN

If you rely on accessing a member of a DN like `member.cn` and have different DN
format, you will have an `Unexpected DN` error. This behaviour is configurable
with the `on_unexpected_dn` key. The possible values are `fail` (the default),
`warn` or `ignore`.


## Examples

``` yaml
- ldap:
    base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
  role:
    name: '{cn}'
    options: LOGIN

- ldap:
    base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
    scope: sub
    filter: >
      (&
        (objectClass=User)
        (memberOf=cn=dba,ou=groups,ou=site,dc=ldap,dc=local)
      )
  roles:
  - names:
    - dba_{member.cn}
    options: LOGIN
    on_unexpected_dn: fail
```
