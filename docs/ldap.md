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

You can also refer to attributes of LDAP entries referenced by the DN with the
same syntax: `<attrname>.<attrname>`. An additional LDAP query will be
performed to retrieve the requested attributes of the entry with the given DN.
For exemple, if a LDAP entry has a `member` attribute with value
`cn=toto,ou=people,dc=ldap,dc=acme,dc=fr` and the LDAP entry with that DN has
a `userPrincipalName` attribute with value `toto@acme.fr`,
`name: '{member.userPrincipalName}'` will generate `toto@acme.fr`.

You can inject attributes in `role:names`, `role:parents`, `role:members` and
`grant:role`.


## Managing heterogeneous DN

If you rely on accessing a member of a DN like `member.cn` and have different DN
format, you will have an `Unexpected DN` error. This behaviour is configurable
with the `on_unexpected_dn` key. The possible values are `fail` (the default),
`warn` or `ignore`.


## Allowing Missing Attributes

The LDAP protocol is loose when querying attributes. A misnamed attributes is
just not returned in the result. The LDAP protocols behave the same way when an
attribute is valid in the schema but undefined in an entry. A *null* attribute
is ambiguous with a nonexistent attribute.

ldap2pg is strict and considers a missing attribute as a misname attribute. You
can however tell ldap2pg that one or more attributes may be missing because
they are optional in LDAP schema. This is the purpose of
`allow_missing_attributes` key.

`allow_missing_attributes` is a list of attribute names. If the LDAP directory
does not return one of these attributes, ldap2pg will default to an empty list.
By default, ldap2pg allows `member` as missing, whatever the returned
objectClass.

``` yaml
sync_map:
- ldap:
    base: ...
    allow_missing_attributes: [member, sAMAccountName]
  roles:
  - name: "{sAMAccountName}"
    members: "{member.cn}"
```


## LDAP Sub-query alias Joins

You may need to issue a sub-query to find more attributes than those available
in Distinguished Name. For example, you may want to query `sAMAccountName` on
Active Directory.

ldap2pg infer sub-query by detecting attributes missing from Distinguished Name.
For example, `member.cn` won't trigger a sub-query while `member.sAMAccountName`
will. The `base` parameter of the sub-query is the value of `member`.

You can adjust sub-query parameters like `scope` and `filter` in the `joins`
paramater. It's a dict with an entry for each sub-query, key is the attribute to
recurse. Each entry is a regular dict with `scope` and `filter` paramater.

!!! notice

   Executing a subquery for each entry of a result set can be very heavy. You
   may optimize the query by using special LDAP query filter like `memberOf`.
   Refer to your LDAP directory documentation and administrator for details.


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
    on_unexpected_dn: fail
  roles:
  - names:
    - dba_{member.cn}
    options: LOGIN
- ldap:
    base: ou=apps,ou=people,dc=ldap,dc=ldap2pg,dc=docker
    scope: sub
    joins:
      member:
        filter: "(objectClass=User)"
    on_unexpected_dn: fail
  roles:
  - names:
    - app_{member.sAMAccountName}
    options: LOGIN
    comment: "App account from LDAP User {member.cn}."
```
