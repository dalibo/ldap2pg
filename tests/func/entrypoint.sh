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

if [ -z "${LDAPBINDDN-}" ] ; then
    exec env $(sed 's/^/LDAP/;s/ \+/=/g' ldaprc) $0 $@
fi

yum_install() {
    local packages=$*
    yum install -y $packages
    rpm --query --queryformat= $packages
}

yum_install epel-release
yum_install \
    cyrus-sasl-md5 \
    make \
    openldap-clients \
    postgresql \
    python \
    python2-pip \
    ${NULL-}

if ! rpm --query --queryformat= ldap2pg ; then
    yum install -y dist/ldap2pg*.noarch.rpm
    rpm --query --queryformat= ldap2pg
fi

# Check Postgres and LDAP connectivity
psql -tc "SELECT version();"
ldapwhoami -xw ${LDAPPASSWORD}

# Install requirements tools with pip.
pip2 install --no-deps --requirement tests/func/requirements.txt

if [ -n "${CI+x}" ] ; then
    # We can't modify config with ldapmodify. This prevent us to setup SASL in
    # CircleCI.
    ldapmodify -xw ${LDAPPASSWORD} -f ./fixtures/openldap-data.ldif
fi

make -C tests/func pytest
