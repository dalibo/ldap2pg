#!/bin/bash -eux

top_srcdir=$(readlink -m $0/../..)
cd "$top_srcdir"
test -f setup.py

uid_gid=$(stat -c %u:%g "$0")
teardown() {
	exit_code=$?
	chown -R "$uid_gid" "$top_srcdir/dist/"
	# If not on CI, wait for user interrupt on exit
	if [ -z "${CI-}" -a $exit_code -gt 0 -a $$ = 1 ] ; then
		tail -f /dev/null
	fi
}
trap teardown EXIT TERM

if rpm --query --queryformat= ldap2pg ; then
    yum remove -y ldap2pg
fi

rm -rf build/bdist*/rpm

rpmdist=$(rpm --eval '%dist')
fullname=$(python setup.py --fullname)
release="${CIRCLE_BUILD_NUM-1}"
requires="python-psycopg2 python-ldap PyYAML"
case $(rpm --eval '%dist') in
	.el6*)
		requires="${requires} python-logutils python-argparse"
		;;
	*)
		;;
esac

# Build it
if ! [ -f "dist/${fullname}.tar.gz" ] ; then
	python setup.py sdist
	release+="snapshot"
fi

python setup.py bdist_rpm \
       --release "${release}%{dist}" \
       --requires "${requires}" \
       --spec-only

chown -R "$(id -u):$(id -g)" "$top_srcdir/dist"
rpmbuild -ba \
	--define "_builddir %{_topdir}" \
	--define "_rpmdir %{_topdir}" \
	--define "_srcrpmdir %{_topdir}" \
	--define "_sourcedir %{_topdir}" \
	--define "_specdir %{_topdir}" \
	--define "_topdir ${top_srcdir}/dist" \
	dist/ldap2pg.spec

rpm="dist/noarch/${fullname}-${release}${rpmdist}.noarch.rpm"
ln -fs "noarch/$(basename $rpm)" dist/ldap2pg-last.rpm

# Test it
yum install -y "$rpm"
cd /
test -x /usr/bin/ldap2pg
python -c 'import ldap2pg'
ldap2pg --version
