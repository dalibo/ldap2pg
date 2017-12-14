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

if [ -z "${LDAPBINDDN-}" ] ; then
    exec env $(sed 's/^/LDAP/;s/ \+/=/g' ldaprc) $0 $@
fi

# Search for the proper RPM package
rpmdist=$(rpm --eval '%dist')
test -f dist/noarch/ldap2pg-*${rpmdist}.noarch.rpm

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
    ${NULL-}

case $rpmdist in
    .el6*)
        yum_install python-pip
        ;;
    *)
        yum_install python2-pip
        ;;
esac

if ! rpm --query --queryformat= ldap2pg ; then
    yum install -y dist/noarch/ldap2pg-*${rpmdist}.noarch.rpm
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
