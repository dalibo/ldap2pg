#!/bin/bash -eux

teardown() {
    # If not on CI, wait for user interrupt on exit
    if [ -z "${CI-}" -a $? -gt 0 -a $$ = 1 ] ; then
        tailf /dev/null
    fi
}

trap teardown EXIT TERM

top_srcdir=$(readlink -m $0/../../..)
cd $top_srcdir
test -f setup.py
test -f dist/ldap2pg-*.noarch.rpm

yum_install() {
    local packages=$*
    yum install -y $packages
    rpm --query --queryformat= $packages
}

yum_install epel-release
yum_install \
    cyrus-sasl-md5 \
    gcc \
    make \
    openldap-clients \
    openldap-devel\
    openssl-devel \
    postgresql \
    python \
    python-devel \
    python2-pip \
    ${NULL-}


# Check Postgres connectivity
psql -tc "SELECT version();"

# Install only ldap2pg and ldap3 package. If other package are required, it's a
# bug.
pip2 install --no-deps pyldap --requirement tests/func/requirements.txt
if ! rpm --query --queryformat= ldap2pg ; then
    yum install -y dist/ldap2pg*.noarch.rpm
    rpm --query --queryformat= ldap2pg
fi

# Looks like ./ldaprc is ignored in CentOS7 by libldap and tools. Put it in envs.
env $(sed 's/^/LDAP/;s/ \+/=/g' ldaprc) pytest tests/func/
