<h1>Querying Directory with LDAP</h1>

ldap2pg reads LDAP searches in `rules` steps in the `ldapsearch` entry.

A LDAP search is **not** mandatory.
ldap2pg can create roles defined statically from YAML.
Each LDAP search is executed once and only once.
There is neither loop nor deduplication of LDAP searches.

!!! tip

    ldap2pg logs LDAP searches as `ldapsearch` commands.
    Enable verbose messages to see them.

    You can debug a failing search by copy-pasting the command in your shell and update parameters.
    Once you are okay, translate back the right parameters in the YAML.


## Configuring Directory Access

ldap2pg reads directory configuration from ldaprc file and LDAP* environment variables.
Known LDAP options are:

- BASE
- BINDDN
- PASSWORD
- REFERRALS
- SASL_AUTHCID
- SASL_AUTHZID
- SASL_MECH
- TIMEOUT
- TLS_REQCERT
- NETWORK_TIMEOUT
- URI

See ldap.conf(5) for the meaning and format of each options.


## Injecting LDAP attributes

Several parameters accepts LDAP attribute injection using curly braces.
To do this, wraps attribute name with curly braces like `{cn}` or `{sAMAccountName}`.
ldap2pg expands to each value of the attribute for each entries of the search.

If the parameter has multiple LDAP attributes,
ldap2pg expands to all combination of attributes for each entries.

Given the following LDAP entries:

``` ldif
dn: uid=dimitri,cn=Users,dc=bridoulou,dc=fr
objectClass: inetOrgPerson
uid: dimitri
sn: Dimitri
cn: dimitri
mail: dimitri@bridoulou.fr
company: external

dn: cn=domitille,cn=Users,dc=bridoulou,dc=fr
objectClass: inetOrgPerson
objectClass: organizationalPerson
objectClass: person
objectClass: top
cn: domitille
sn: Domitille
company: acme
company: external
```

The format `{company}_{cn}` with the above LDAP entries generates the following strings:

- `acme_domitille`
- `external_domitille`
- `external_dimitri`

The pseudo attribute `dn` is always available and references the Distinguished Name of the original LDAP entry.


### Accessing RDN and sub-search

If an attribute type is Distinguished Name (DN),
you can refer to a Relative Distinguished Name (RDN) with a dot, like this: `<attribute>.<rdn>`.
If an RDN has multiple values, only the first value is returned.
There is no way to access other value.

For example,
if a LDAP entry has `member` attribute with value `cn=toto,cn=Users,dc=bridoulou,dc=fr`,
the `{member.cn}` format will generate `toto`.
The `{member.dc}` format will generate `ldap`.
There is no way to access `acme` and `fr`.

Known RDN are `cn`, `l`, `st`, `o`, `ou`, `c`, `street`, `dc`, and `uid`.
Other attributes triggers a sub-search.
The format `{member.sAMAccountName}` will issue a sub-search for all `member` value as LDAP search base narrowed to `sAMAccountName` attribute.


### LDAP Attribute Case

When injecting an LDAP attribute with curly braces,
you can control the case of the value using `.lower()` or `.upper()` methods.

``` yaml
- ldapsearch: ...
  role: "{cn.lower()}"
```
