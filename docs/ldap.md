<h1>Querying Directory with LDAP</h1>

ldap2pg reads LDAP searches in `sync_map` items in the `ldapsearch` entry.

A LDAP search is **not** mandatory. ldap2pg can create roles defined statically
from YAML. Each LDAP search is executed once and only once. There is neither
loop nor deduplication of LDAP searches.

!!! tip

    ldap2pg logs LDAP searches as `ldapsearch` commands. Enable verbose messages to
    see them.

    You can debug a failing search by copy-pasting the command in your
    shell and update parameters. Once you are okay, translate back the right
    parameters in the YAML.


## Injecting LDAP attributes

Several parameters accepts LDAP attribute injection using curly braces. To do
this, wraps attribute name with curly braces like `{cn}` or `{sAMAccountName}`.
ldap2pg expands to each value of the attribute for each entries of the search.

If the parameter has multiple LDAP attributes, ldap2pg expands to all
combination of attributes for each entries. Given the following LDAP entries:

``` ldif
dn: uid=dimitri,ou=people,dc=ldap,dc=acme,dc=tld
objectClass: inetOrgPerson
uid: dimitri
sn: Dimitri
cn: dimitri
mail: dimitri@ldap2pg.docker
company: external

dn: cn=domitille,ou=people,dc=ldap,dc=acme,dc=tld
objectClass: inetOrgPerson
objectClass: organizationalPerson
objectClass: person
objectClass: top
cn: domitille
sn: Domitille
company: acme
company: external
```

The format `{company}_{cn}` with the above LDAP entries generates the following
strings:

- `acme_domitille`
- `external_domitille`
- `external_dimitri`

The pseudo attribute `dn` is always available and references the Distinguished
Name of the original LDAP entry.


### Accessing RDN and sub-search

If an attribute type is Distinguished Name (DN), you can refer to a Relative
Distinguished Name (RDN) with a dot, like this: `<attribute>.<rdn>`. If an RDN
has multiple values, only the first value is returned. There is no way to
access other value.

For example, if a LDAP entry has `member` attribute with value
`cn=toto,ou=people,dc=ldap,dc=acme,dc=fr`, the `{member.cn}` format will
generate `toto`. The `{member.dc}` format will generate `ldap`. There is no way
to access `acme` and `fr`.

Known RDN are `cn`, `l`, `st`, `o`, `ou`, `c`, `street`, `dc`, and `uid`. Other
attributes triggers a sub-search. The format `{member.sAMAccountName}` will
issue a sub-search for all `member` value as LDAP search base narrowed to
`sAMAccountName` attribute.


### LDAP Attribute Case

When injecting an LDAP attribute with curly braces, you can control the case of
the value using `.lower()` or `.upper()` methods.

``` yaml
- ldapsearch: ...
  role: "{cn.lower()}"
```

ldap2pg will try to rename a role when case is changing, instead of dropping
and creating. ldap2pg will rename only if there is no doubt. For example,
ldap2pg refuses to choose between `ALICE` and `alice` to be renamed to `Alice`.
On the other way around, if an existing role `Alice` is existing and both
`alice` and `ALICE` are wanted, `Alice` will be dropped instead of renamed.

ldap2pg still accepts typo squatting. If you want both `Alice` and `ALICE`,
ldap2pg won't confuse between them.


## Examples

The following example creates a Postgres role for each entry sub
`ou=people,dc=ldap,dc=ldap2pg,dc=docker` with `LOGIN` option.

``` yaml
- ldapsearch:
    base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
  role:
    name: '{cn}'
    options: LOGIN
```

The following example uses the `memberOf` extension in a custom LDAP filter to
get `User` member of the group `dba`. The name of the generated Postgres roles
is prefixed by `dba_`.

``` yaml
- ldapsearch:
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
    - dba_{cn}
    options: LOGIN SUPERUSER
```

The following example issues a sub-search to fetch `sAMAccountName` attribute.
A unique comment is generated for each role using `member` DN.

``` yaml
- ldapsearch:
    base: ou=apps,ou=people,dc=ldap,dc=ldap2pg,dc=docker
    scope: sub
    joins:
      member:
        filter: "(objectClass=Group)"
    on_unexpected_dn: fail
  roles:
  - names:
    - app_{member.sAMAccountName}
    options: LOGIN
    comment: "From LDAP entry {member}, member of {dn}."
```


## Forcing Simple Bind

By default, OpenLDAP utils uses SASL and use must explicitly use `-x` CLI
switch to force simple bind authentication. ldap2pg has a different behaviour.
ldap2pg does not have default SASL mechanism. If `SASL_MECH` is empty or
undefined, ldap2pg uses simple bind.

If you want to force simple bind, ensure `SASL_MECH` is none.
