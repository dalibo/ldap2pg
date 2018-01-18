<h1>Querying with LDAP</h1>

`ldap2pg` searches for LDAP query in `sync_map` items in the `ldap` entry.

A `ldap` entry contains `base`, `scope` and `filter`. The meaning of `base`,
`scope` and `filter` is strictly the same as in `ldapsearch`. `base` must be
defined. `filter` defaults to `(objectClass=*)`, just like `ldapsearch(1)`.
`scope` defaults to `sub`, likewise. `ldap2pg` determine which attribute to
query depending on `_attribute` values in `grant` and `roles` rules.

!!! tip

    `ldap2pg` logs LDAP queries as `ldapsearch` commands. You can debug a failing
    query by copy-pasting the command in your shell and update parameters. Once you
    are okay, report back the right parameters.

A LDAP search is **not** mandatory. `ldap2pg` can create roles defined
statically from YAML.

Each LDAP search is done once and only once. There is neither loop nor
deduplication of LDAP searches.


## Mapping LDAP attributes to Postgres

In `role` and `grant` rules, each key ending with `_attribute` maps to an
attribute in the LDAP entry.

`ldap2pg` apply the `role` or `grant` rule for each value of the attribute of
each entries returned by the directory.

If the attribute is of type Distinguished Name (DN), you can refer to the first
Relative Distinguished Name (RDN) of the value like this: `attrname.rdn`. For
example, if a LDAP entry has `member` attribute with value
`cn=toto,ou=people,dc=ldap,dc=acme,dc=fr`, `name_attribute: member.cn` will
generate `toto`.


## Examples

``` yaml
- ldap:
    base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
  role:
    name_attribute: cn
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
  - name_attribute: member.cn
    options: LOGIN
```
