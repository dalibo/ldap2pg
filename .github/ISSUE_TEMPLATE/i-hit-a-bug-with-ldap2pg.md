---
name: I hit a bug with ldap2pg
about: Report a bug with detailed information
title: ''
labels: ''
assignees: ''

---

<!--

Hi ! Thanks for reporting to us !

If you encounter a bug in ldap2pg, would you mind to paste the following
informations in issue description:

**IMPORTANT**: Please DO NOT publish personal data or confidential information
in this issue. We are responsible to delete any sensitive data in this project.

French accepted.

-->


## ldap2pg.yml

<!-- Ensure there is no password ! -->

<details><summary>ldap2pg.yml</summary>
``` yaml
postgres:
  ...

rules:
  ...
```
</details>


## Expectations

- What you expected from ldap2pg ?
- What ldap2pg did wrong ?


## Verbose output of ldap2pg execution

<details><summary>Verbose output</summary>
``` console
$ ldap2pg --verbose --real
[ldap2pg.config        INFO] Starting ldap2pg ...
...
$
```
</details>
