#!/bin/bash -eux

teardown() {
	# If not on CI, wait for user interrupt on exit
	if [ -z "${CI-}" -a "$?" -gt 0 -a $$ = 1 ] ; then
		sleep infinity
	fi
}

trap teardown EXIT TERM

top_srcdir=$(readlink -m $0/../../..)
cd $top_srcdir
test -f setup.py

# Choose target Python version. Matches packaging/rpm/build_rpm.sh.
rpmdist=$(rpm --eval '%dist')
case "$rpmdist" in
	*.el6|*.el7)
		python=python2
		pip=pip2
		;;
	*.el8)
		python=python3
		pip=pip3
		;;
esac
fullname=$($python setup.py --fullname)

# Search for the proper RPM package.
rpms=(dist/noarch/${fullname}-*${rpmdist}.noarch.rpm)
rpm=${rpms[0]}
test -f "$rpm"

# Clean and install package.
if rpm --query --queryformat= ldap2pg ; then
	yum -q -y remove ldap2pg
fi

yum -q -y install "$rpm"
ldap2pg --version

# Check Postgres and LDAP connectivity
psql -tc "SELECT version();"
# ldap-utils on CentOS does not read properly current ldaprc. Linking it in ~
# workaround this.
ln -fsv "${PWD}/ldaprc" ~/ldaprc
ldapwhoami -x -d 1 -w "${LDAPPASSWORD}"

"$pip" install --prefix=/usr/local --requirement tests/func/requirements.txt

if [ -n "${CI+x}" ] ; then
    # We can't modify config with ldapmodify. This prevent us to setup SASL in
    # CircleCI.
    ldapmodify -xw "${LDAPPASSWORD}" -f ./fixtures/openldap-data.ldif
fi

pytest -x tests/func/
