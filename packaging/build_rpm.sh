#!/bin/bash -eux

top_srcdir=$(readlink -m $0/../..)
cd "$top_srcdir"
test -f setup.py

uid_gid=$(stat -c %u:%g "$0")
teardown() {
	exit_code=$?
	# If not on CI, wait for user interrupt on exit
	if [ -z "${CI-}" ] && [ $exit_code -gt 0 ] && [ $$ = 1 ] ; then
		tail -f /dev/null
	fi
}
trap teardown EXIT TERM

if rpm --query --queryformat= ldap2pg ; then
    yum remove -y ldap2pg
fi

rm -rf build/bdist*/rpm


#       S O U R C E S

rpmdist=$(rpm --eval '%dist')
version=$(grep -Po "version='\K\d.\d" setup.py)
tarball="dist/ldap2pg-$version.tar.gz"
spec="packaging/ldap2pg-${rpmdist#.}.spec"

# Install builddep asap to have Python.
yum-builddep -y "$spec"

# Build dist for CI.
if ! [ -f "$tarball" ] ; then
	python=$(grep -Po 'BuildRequires: \Kpython\d' "$spec")
	$python setup.py sdist
	test -f "$tarball"
fi

topdir=~testuser/rpmbuild
mkdir -p "$topdir/SOURCES" "$topdir/SPECS"
cp -vf "$spec" "$topdir/SPECS/ldap2pg.spec"
sed -i "/^%define.\\+version/s/.\\..\\+/$version/" "$topdir/SPECS/ldap2pg.spec"
grep -F "version $version" "$_"
cp -vf "$tarball" "$topdir/SOURCES/"
# rpmbuild requires files to be owned by running uid
chown -R testuser "$topdir"


#       B U I L D


sudo -u testuser rpmbuild \
	--define "_topdir $topdir" \
	-bb "$topdir/SPECS/ldap2pg.spec"

rpm="$topdir/RPMS/noarch/ldap2pg-${version}-1${rpmdist}.noarch.rpm"
test -f "$rpm"
cp "$rpm" dist/
rpm="${rpm##*/}"
ln -fs "$rpm" dist/ldap2pg-last.rpm
chown --no-dereference "$uid_gid" "dist/$rpm" dist/ldap2pg-last.rpm


#       P E N   T E S T

yum install -y "dist/$rpm"
cd /
test -x /usr/bin/ldap2pg
ldap2pg --version
