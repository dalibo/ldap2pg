<h1>Installation</h1>

`ldap2pg` main packaging format is a regular Python package, available at PyPI.
`ldap2pg` tries to reduce dependencies and to be compatible with versions
available from official distributions repositories.

# Pure python

You can fetch all dependencies with PIP. Choose either `pip3` or `pip2`.

``` console
# apt install -y libldap2-dev libsasl2-dev python3-pip
# pip3 install ldap2pg psycopg2-binary
```

# On CentOS 8 from RPM

On CentOS 8, either [PGPG YUM repository](https://yum.postgresql.org/) and
[Dalibo Labs YUM repository](https://yum.dalibo.org/labs/) offer RPM package
for ldap2pg. The repositories does not provide the same packaging. Dalibo Labs
is upstream, packages are more up to date. PGDG is more common and often always
available on your host.

For using Dalibo Labs packaging:

- [Enable Dalibo Labs YUM repository](https://yum.dalibo.org/labs/).
- Install `ldap2pg` package with yum:

``` console
# yum install ldap2pg
...
# ldap2pg --version
```

For using PGDG YUM packaging:

- [Enable PGDG YUM repository](https://yum.postgresql.org/).
- Install `python3-ldap2pg`.


# On CentOS 7 from RPM

On CentOS 7, choose either Dalibo Labs repository or PGDG like for CentOS 8.
Note that Dalibo package uses Python 2.7 while PGDG ships only Python 3
package.

For using Dalibo Labs packaging:

- [Enable Dalibo Labs YUM repository](https://yum.dalibo.org/labs/).
- Install `ldap2pg` package with yum:

``` console
# yum install ldap2pg
...
# ldap2pg --version
```

For using PGDG YUM packaging:

- [Enable PGDG YUM repository](https://yum.postgresql.org/).
- Install `python3-ldap2pg`.


# On CentOS 7 from source

You should run `ldap2pg` with Python2.7 to use packaged dependencies.

``` console
# yum install -y epel-release
# yum install -y python2-pip python-ldap python-psycopg2 python-wheel PyYAML
# pip2 install --no-deps --upgrade ldap2pg
# ldap2pg --version
```


# On CentOS 6

PGDG repository provides RPM packages for CentOS6.

To install from source, you have to run `ldap2pg` with Python2.6 and some
forward compatibility dependencies.

``` console
# yum install -y epel-release
# yum install -y pyton-argparse python-pip python-ldap python-logutils python-psycopg2 PyYAML
# pip install --no-deps --upgrade ldap2pg
# ldap2pg --version
```


# On Debian 9 (jessie)

On Debian jessie or later, you can use regular Python3 and wheels.

``` console
# apt install -y python3-pip python3-psycopg2 python3-yaml
# pip3 install --no-deps ldap2pg python-ldap
```

# On Debian 7 (wheezy)

On Debian wheezy, you have to use Python2.7.

``` console
# apt install -y python-pip python-psycopg2 python-yaml python-ldap
# pip install --no-deps ldap2pg
```
