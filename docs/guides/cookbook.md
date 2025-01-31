---
hide:
  - navigation
---

<h1>Cookbook</h1>

Here in this cookbook, you'll find some recipes for various use case of
ldap2pg.

If you struggle to find a way to setup ldap2pg for your needs,
please [open a discussion](https://github.com/dalibo/ldap2pg/discussions) so that we can update
*Cookbook* with new recipes!
Your contribution is welcome!


## Configure `pg_hba.conf` with LDAP

ldap2pg does **NOT** configure PostgreSQL for you. You should carefully read
[PostgreSQL LDAP documentation] for this point. Having PostgreSQL properly
configured **before** writing `ldap2pg.yaml` is a good start. Here is the steps
to setup PostgreSQL with LDAP in the best order:

- Write the LDAP search and test it with `ldapsearch(1)`. This way, you can
  also check how you connect to your LDAP directory.
- In PostgreSQL cluster, **manually** create a single role having its password
  in LDAP directory.
- Edit `pg_hba.conf` following [PostgreSQL LDAP documentation] until you can
  effectively login with the single role and the password from LDAP.

[PostgreSQL LDAP documentation]: https://www.postgresql.org/docs/current/static/auth-methods.html#auth-ldap

Once you have LDAP authentication configured in PostgreSQL cluster, you can
move to automate role creation from the LDAP directory using ldap2pg:

- Write a simple `ldap2pg.yaml` with only one LDAP search just to setup ldap2pg
  connection parameters for PostgreSQL and LDAP connection. ldap2pg always run
  in dry mode by default, so you can safely loop ldap2pg execution until you
  get it right.
- Then, complete `ldap2pg.yaml` to fit your needs following [ldap2pg
  documentation](../cli.md). Run ldap2pg for real and check that ldap2pg maintain
  your single test role, and that you can still connect to the cluster with it.
- Finally, you must decide when and how you want to trigger synchronization: a
  regular cron tab ? An ansible task ? Manually ? Other ? Ensure ldap2pg
  execution is frequent, on purpose and notified.


## Search LDAP Directory

The first step is to search your LDAP server with ldapsearch(1), the CLI tool
from OpenLDAP. Like this:

``` console
$ ldapsearch -H ldaps://ldap.ldap2pg.docker -U testsasl -W
Enter LDAP Password:
SASL/DIGEST-MD5 authentication started
SASL username: testsasl
SASL SSF: 128
SASL data security layer installed.
# extended LDIF
#
# LDAPv3
...
# search result
search: 4
result: 0 Success

# numResponses: 16
# numEntries: 15
$
```

Now save the settings in `ldaprc`:

``` yaml
LDAPURI ldaps://ldap.ldap2pg.docker
LDAPSASL_AUTHCID testsasl
```

And in environment: `LDAPPASSWORD=secret`

Next, update your ldapsearch(1) to properly match role entries in LDAP server:

``` console
$ ldapsearch -H ldaps://ldap.ldap2pg.docker -U testsasl -W -b cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker '' member
...
# dba, groups, ldap.ldap2pg.docker
dn: cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker
member: cn=Alan,ou=people,dc=ldap,dc=ldap2pg,dc=docker
member: cn=albert,ou=people,dc=ldap,dc=ldap2pg,dc=docker
member: cn=ALICE,ou=people,dc=ldap,dc=ldap2pg,dc=docker

# search result
search: 4
result: 0 Success

...
$
```

Now translate the query in `ldap2pg.yaml` and associate a role mapping to
produce roles from each values of each entries returned by the LDAP search:

``` yaml
- ldapsearch:
    base: cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker
  role:
    name: '{member.cn}'
    options: LOGIN SUPERUSER
```

Test it:

``` console
$ ldap2pg
...
Querying LDAP cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker...
Would create alan.
Would create albert.
Would update options of alice.
...
Comparison complete.
$
```

Read further on how to control role creation from LDAP entry in
[Configuration](../config.md). Once you're satisfied with the comparison output, go
real with `--real`.


## Using LDAP High-Availability

ldap2pg supports LDAP High Availability out of the box just like any openldap client.
Use a space separated list of URI to tells all servers.

``` console
$ LDAPURI="ldaps://ldap1 ldaps://ldap2" ldap2pg
```

See [ldap.conf(5)] for further details.

[ldapLconf(5)]: https://www.openldap.org/software/man.cgi?query=ldap.conf


## Running as non-superuser

Since Postgres provide a `CREATEROLE` role option, you can manage roles without superuser privileges.
Security-wise, it's a good idea to manage roles without super privileges.

!!! warning

    Up to Postgres 15, having CREATEROLE is roughly equivalent to being superuser.
    This because CREATEROLE user can grant themselve almost every privileges.
    Thus ldap2pg supports running unprivileged against Postgres 16 and later only.

ldap2pg supports this case.
However, you must be careful about the limitations.
Let's call the non-super role creating other roles `creator`.

- You can't manage some roles options like `SUPERUSER`, `BYPASSRLS` and `REPLICATION`.
  Thus you wont be able to detect spurious superusers.
- Ensure `creator` can revoke all grants of managed users.


## Removing All Roles

If ever you want to clean all roles in a PostgreSQL cluster, ldap2pg could be
helpful. You must explicitly define an empty `rules`.

``` console
$ echo '{version: 6, rules: []}' | ldap2pg --config -
...
Empty synchronization map. All roles will be dropped!
...
```

In this example, default blacklist applies.
ldap2pg never drop its connection role.


## ldap2pg as Docker container

Already familiar with Docker and willing to save the setup time? You're at the
right place.

To run the container simply use the command:
``` console
$ docker run --rm dalibo/ldap2pg --help
```

The Docker image of ldap2pg use the same configuration options as explained in the [cli](../cli.md) and [ldap2pg.yml](../config.md) sections.
You can mount the ldap2pg.yml configuration file.

``` console
$ docker run --rm -v ${PWD}/ldap2pg.yml:/workspace/ldap2pg.yml dalibo/ldap2pg
```

You can also export some environmnent variables with the **-e** option:

``` console
$ docker run --rm -v ${PWD}/ldap2pg.yml:/workspace/ldap2pg.yml -e PGDSN=postgres://postgres@localhost:5432/ -e LDAPURI=ldaps://localhost -e LDAPBINDDN=cn=you,dc=entreprise,dc=fr -e LDAPPASSWORD=pasglop dalibo/ldap2pg
```

Make sure your container can resolve the hostname your pointing to.
If you use some internal name resolution be sure to add the **--dns=** option to your command pointing to your internal DNS server.
More [info](https://docs.docker.com/v17.09/engine/userguide/networking/default_network/configure-dns/)
