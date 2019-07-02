---
name: Bug report
about: Report a bug in ldap2pg
title: ''
labels: ''
assignees: ''

---

<!--

Hi ! Thanks for reporting to us !

If you encounter a bug in ldap2pg, would you mind to paste the following
informations in issue description:

-->


## ldap2pg.yml

<!-- Ensure there is no password ! -->

``` yaml
# ldap2pg.yml
postgres:
  ...

sync_map:
  ...
```

## Expectations

- What you expected from ldap2pg ?
- What ldap2pg did wrong ?


## Verbose output of ldap2pg execution

``` console
$ ldap2pg --verbose -N
[ldap2pg.config        INFO] Starting ldap2pg ...
...
$
```
