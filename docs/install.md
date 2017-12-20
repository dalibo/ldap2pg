<h1>Installation</h1>

`ldap2pg` main packaging format is a regular Python package, available at PyPI.
`ldap2pg` tries to reduce dependencies and to be compatible with versions
available from official distributions repositories.

# Pure python

You can fetch all dependencies with PIP. Choose either `pip3` or `pip2`.

``` console
# apt install -y libldap2-dev libsasl2-dev python3-pip
# pip3 install ldap2pg
```

# On CentOS 7

On CentOS 7, you should run `ldap2pg` with Python2.7 to use packaged
dependencies.

``` console
# yum install -y epel-release
# yum install -y python2-pip python-ldap python-psycopg2 python-wheel PyYAML
# pip2 install --no-deps --upgrade ldap2pg
# ldap2pg --version
```


# On Debian

On Debian jessie or later, you can use regular Python3 and wheel package.

``` console
# apt install -y python3-pip python3-psycopg2 python3-yaml python3-pyldap
# pip3 install --no-deps ldap2pg
```
