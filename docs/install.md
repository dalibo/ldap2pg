<h1>Installation</h1>


## Requirements

ldap2pg is released as a single binary with no dependencies.

On runtime, ldap2pg requires an unprivileged role with `CREATEDB` and `CREATEROLE` options or a superuser access.
ldap2pg does not require to run on the same host as the synchronized PostgreSQL cluster.

With 2MiB of RAM and one vCPU, ldap2pg can synchronize several thousands of roles in seconds,
depending on PostgreSQL instance and LDAP directory response time.


## Installing

/// tab | Debian/Alpine

Download package for your target system and architecture from [ldap2pg release page].

///

/// tab | YUM/DNF

On RHEL and compatible clone, [Dalibo Labs YUM repository](https://yum.dalibo.org/labs/) offer RPM package for ldap2pg.

For using Dalibo Labs packaging:

- [Enable Dalibo Labs YUM repository](https://yum.dalibo.org/labs/).
- Install `ldap2pg` package with yum:
  ```
  yum install ldap2pg
  ```

///

/// tab | Manual

- Download binary for your target system and architecture from [ldap2pg release page].
- Move the binary to `/usr/local/bin`.
- Ensure it's executable with `chmod 0755 /usr/local/bin/ldap2pg`.
- Test installation with `ldap2pg --version`.

///

[ldap2pg release page]: https://github.com/dalibo/ldap2pg/releases
