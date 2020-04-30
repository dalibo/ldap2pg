#!/bin/bash -eux

teardown() {
	exit_code=$?
	# If not on CI, wait for user interrupt on exit
	if [ -z "${CI-}" -a $exit_code -gt 0 -a $$ = 1 ] ; then
		tail -f /dev/null
	fi
}

trap teardown EXIT TERM

top_srcdir=$(readlink -m $0/../..)
cd $top_srcdir
test -f setup.py

yum_install() {
    local packages=$*
    sudo yum install -y $packages
    rpm --query --queryformat= $packages
}

# Fasten yum by disabling updates repository
sudo sed -i '/^\[updates\]/,/^gpgkey=/d' /etc/yum.repos.d/CentOS-Base.repo
yum_install epel-release
yum_install python-setuptools

if rpm --query --queryformat= ldap2pg ; then
    sudo yum remove -y ldap2pg
fi

rm -rf build/bdist*/rpm

rpmdist=$(rpm --eval '%dist')
requires="python-psycopg2 python-ldap PyYAML"
case $(rpm --eval '%dist') in
	.el6*)
		requires="${requires} python-logutils python-argparse"
		;;
	*)
		;;
esac

# Build it
python setup.py sdist bdist_rpm \
       --release ${CIRCLE_BUILD_NUM-1}%{dist} \
       --requires "${requires}" \
       --spec-only

rpmbuild -ba \
	--define "_topdir ${top_srcdir}/dist" \
	--define "_sourcedir ${top_srcdir}/dist" \
	dist/ldap2pg.spec

# Test it
sudo yum install -y dist/noarch/ldap2pg*${rpmdist}.noarch.rpm

test -x /usr/bin/ldap2pg
python -c 'import ldap2pg'
ldap2pg --help
