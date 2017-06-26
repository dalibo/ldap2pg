#!/bin/bash -eux

teardown() {
    # If not on CI, wait for user interrupt on exit
    if [ -z "${CI-}" -a $? -gt 0 ] ; then
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
yum_install python python2-pip postgresql openldap-clients

# Check PostgreSQL access and create fixture
createuser spurious

# Check OpenLDAP access and load fixtures
ldapadd -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD -f dev-fixture.ldif

# Install only ldap2pg and ldap3 package. If other package are required, it's a
# bug.
pip2 install --no-deps ldap3
yum install -y dist/ldap2pg*.noarch.rpm
rpm --query --queryformat= ldap2pg

# Case dry run
DEBUG=1 DRY=1 ldap2pg
# Assert nothing is done
psql -c 'SELECT rolname FROM pg_roles;' | grep -q spurious

# Case real run
DEBUG=1 ldap2pg

# Assert spurious role is dropped
! psql -c 'SELECT rolname FROM pg_roles;' | grep -q spurious
test ${PIPESTATUS[0]} -eq 0

psql -c 'SELECT rolname FROM pg_roles;' | grep -q alice
psql -c 'SELECT rolname FROM pg_roles;' | grep -q bob
