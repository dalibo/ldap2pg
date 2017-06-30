<h1>Installation</h1>

`ldap2pg` main packaging format is a regular Python package, available at PyPI.
`ldap2pg` tries to reduce dependencies and to be compatible with versions
available from official distributions repositories.

# Pure python

You can fetch all dependencies with PIP. Choose either `pip3` or `pip2`.

``` console
# apt install libpq-dev python3-pip python3-wheel
# pip3 install ldap2pg
```

# On CentOS 7

On CentOS 7, you should run `ldap2pg` with Python2.7 to use packaged
dependencies.

``` console
# yum install -y epel-release
# yum install -y python python2-pip python-six python-psycopg2 PyYAML python2-pyasn1
# pip2 install --no-deps ldap2pg ldap3 
```


# On Debian

On Debian jessie, you can use regular Python3.4.

``` console
# apt install -y python3-pip python3-psycopg2 python3-six python3-wheel python3-yaml
# pip3 install --no-deps ldap2pg ldap3
```
