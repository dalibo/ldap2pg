---
hide:
  - navigation
---

<h1>Installation</h1>

ldap2pg main packaging format is a regular Python package, available at PyPI as
source and binary wheel. ldap2pg tries to reduce dependencies and to be
compatible with versions available from official distributions repositories.


## Requirements

ldap2pg requires Python 2.6+ or Python 3.4+, pyyaml, python-ldap and
python-psycopg2. ldap2pg is well tested on Linux.

ldap2pg recommends to use your distribution packages for dependencies and for
ldap2pg if available.

On runtime, ldap2pg requires a superuser access or a role with `CREATEROLE`
option. ldap2pg does not require to run on the same host as the PostgreSQL
cluster synchronized.


## On RHEL 8/7/6 from RPM

On RHEL and compatible clone, either [PGPG YUM
repository](https://yum.postgresql.org/) and [Dalibo Labs YUM
repository](https://yum.dalibo.org/labs/) offer RPM package for ldap2pg. Each
repository does not provide the same packaging. Dalibo Labs is upstream,
packages are more up to date. PGDG is more common and has more chances to be
available on your host.

For using Dalibo Labs packaging:

- [Enable Dalibo Labs YUM repository](https://yum.dalibo.org/labs/).
- Install `ldap2pg` package with yum:
  ```
  yum install ldap2pg
  ```

For using PGDG YUM packaging:

- [Enable PGDG YUM repository](https://yum.postgresql.org/).
- Install `python3-ldap2pg`.
  ```
  yum install python3-ldap2pg
  ```


## On RHEL 7 from pip

You should run `ldap2pg` with Python3.6 to use RHEL packaged dependencies.

- Install EPEL:
  ```
  yum install -y epel-release
  ```
- Install dependencies:
  ```
  yum install -y python36-pip python36-ldap python36-psycopg2 python36-wheel python36-PyYAML
  ```
- Now install ldap2pg with pip:
  ```
  pip3 install --no-deps --upgrade ldap2pg
  ```
- Check installation with:
  ```
  ldap2pg --version
  ```


## On RHEL 6 from pip

On RHEL 6, pip-2.6 can't access PyPI anymore. PyPI uses CloudFlare which
requires the SNI TLS extension not available in Python 2.6 SSL. You must
download each dependencies manually and install them locally using pip. You'd
better use RPM.


## On Debian 11 (bookworm) / 10 (buster)

On Debian buster and bookworm, you can use regular Python3 and wheels.

- Install dependencies from Debian repositories:
  ```
  apt install -y --no-install-recommends python3-pip python3-ldap python3-pkg-resources python3-psycopg2 python3-yaml
  ```
- Install ldap2pg from PyPI:
  ```
  pip3 install --no-deps ldap2pg
  ```
- Check installation:
  ```
  ldap2pg --version
  ```


## On Debian 9 (stretch)

On Debian stretch, you can use regular Python3 and wheels.

- Install dependencies from Debian repositories:
  ```
  apt install -y --no-install-recommends build-essential libldap2-dev libsasl2-dev python3-dev python3-pip python3-psycopg2 python3-pyasn1-modules python3-setuptools python3-wheel python3-yaml
  ```
- Install ldap2pg and python-ldap for Python 3.5.
  ```
  pip3 install --no-deps ldap2pg "python-ldap<3.4"
  ```
- Check installation:
  ```
  ldap2pg --version
  ```


## On Debian 8 (jessie)

On Debian jessie, you can use regular Python3 and wheels.

- Install dependencies from Debian repositories:
  ```
  apt install -y --no-install-recommends libldap2-dev libsasl2-dev python3-dev python3-pip build-essential python3-psycopg2 python3-yaml python3-pyasn1-modules
  ```
- Install ldap2pg and python-ldap for Python 3.4.
  ```
  pip3 install --no-deps ldap2pg "python-ldap<3.4"
  ```
- Check installation:
  ```
  ldap2pg --version
  ```


## Using pip

You can fetch all dependencies with pip. You must select which psycopg2 package
you want. To build python-ldap you need python, libldap2 and libsasl2
development files and a compiler.

```
pip3 install ldap2pg psycopg2-binary
```
