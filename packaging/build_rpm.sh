#!/bin/bash -eux

top_srcdir=$(readlink -m $0/../..)
cd $top_srcdir
test -f setup.py

yum_install() {
    local packages=$*
    yum install -y $packages
    rpm --query --queryformat= $packages
}

yum_install epel-release
yum_install python python2-pip rpm-build
pip install -U pip setuptools

if rpm --query --queryformat= ldap2pg ; then
    yum remove -y ldap2pg
fi

rm -rf build/bdist*/rpm
# Build it
python setup.py bdist_rpm --release ${CIRCLE_BUILD_NUM-1}

# Test it
yum install -y dist/ldap2pg*.noarch.rpm

test -x /usr/bin/ldap2pg
python -c 'import ldap2pg'

chown --changes --recursive $(stat -c %u:%g setup.py) dist/ build/
