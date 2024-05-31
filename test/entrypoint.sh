#!/bin/bash -eux

teardown() {
	# If not on CI, wait for user interrupt on exit
	if [ -z "${CI-}" -a "$?" -gt 0 -a $$ = 1 ] ; then
		sleep infinity
	fi
}

trap teardown EXIT TERM

top_srcdir=$(readlink -m "$0/../..")
cd "$top_srcdir"
test -f go.mod

export LC_ALL=en_US.utf8

# Choose target Python version. Matches packaging/rpm/build_rpm.sh.
rpmdist=$(rpm --eval '%dist')
case "$rpmdist" in
	*.el9)
		python=python3.9
		pip=pip3.9
		;;
	*.el7|*.el8)
		python=python3.6
		pip=pip3.6
		;;
	*.el6)
		python=python2
		pip=pip2
		;;
esac

"$pip" --version
if "$pip" --version |& grep -Fiq "python 2.6" ; then
	pip26-install https://files.pythonhosted.org/packages/53/67/9620edf7803ab867b175e4fd23c7b8bd8eba11cb761514dcd2e726ef07da/py-1.4.34-py2.py3-none-any.whl
	pip26-install https://files.pythonhosted.org/packages/fd/3e/d326a05d083481746a769fc051ae8d25f574ef140ad4fe7f809a2b63c0f0/pytest-3.1.3-py2.py3-none-any.whl
	pip26-install https://files.pythonhosted.org/packages/86/84/6bd1384196a6871a9108157ec934a1e1ee0078582cd208b43352566a86dc/pytest_catchlog-1.2.2-py2.py3-none-any.whl
	pip26-install https://files.pythonhosted.org/packages/4a/22/17b22ef5b049f12080f5815c41bf94de3c229217609e469001a8f80c1b3d/sh-1.12.14-py2.py3-none-any.whl
else
	"$pip" install --prefix=/usr/local --requirement test/requirements.txt
fi

# Check Postgres and LDAP connectivity
psql -tc "SELECT version();"
# ldap-utils on CentOS does not read properly current ldaprc. Linking it in ~
# workaround this.
ln -fsv "${PWD}/ldaprc" ~/ldaprc
retry ldapsearch -x -v -w "${LDAPPASSWORD}" -z none >/dev/null

export KRB5_CONFIG="${PWD}/test/krb5.conf"
kinit -V -k -t "${PWD}/test/samba.keytab" Administrator
LDAPURI="${LDAPURI/ldaps:/ldap:}" ldapsearch -v -Y GSSAPI -U Administrator >/dev/null
"$python" -m pytest test/ "$@"
