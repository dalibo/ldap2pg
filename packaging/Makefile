TOPSRC_DIR=$(wildcard ..)
YUM_LABS?=$(wildcard $(TOPSRC_DIR)/../yum-labs)

default:

sdist:
	pip download --no-deps --no-binary :all: --dest $(TOPSRC_DIR)/dist ldap2pg==$(shell cd ..; python setup.py --version)

distclean:
	rm -rf ../dist/

rpm:
	$(MAKE) -C $(YUM_LABS) clean
	$(MAKE) rpm-centos8 rpm-centos7 rpm-centos6

rpm-centos%:
	DIST=centos$* docker-compose run --rm rpm
	cp -fl $$(readlink -e $(TOPSRC_DIR)/dist/ldap2pg-last.rpm) $(YUM_LABS)/rpms/CentOS$*-x86_64/

clean:
	docker-compose down -v

push:
	@make -C $(YUM_LABS) push createrepos clean

specs: ldap2pg-el8.spec ldap2pg-el7.spec ldap2pg-el6.spec

ldap2pg-el8.spec: Makefile ../setup.py
	cd ..; python setup.py bdist_rpm \
	  --release "1%{dist}" \
	  --build-requires "python3 python3-setuptools" \
	  --requires "python3-psycopg2 python3-ldap python3-yaml" \
	  --python "/usr/bin/python3" \
	  --spec-only
	cp -f ../dist/ldap2pg.spec $@

ldap2pg-el7.spec: Makefile ../setup.py
	cd ..; python setup.py bdist_rpm \
	  --release "1%{dist}" \
	  --build-requires "python2 python2-setuptools" \
	  --requires "python-psycopg2 python-ldap PyYAML" \
	  --python "/usr/bin/python2" \
	  --spec-only
	cp -f ../dist/ldap2pg.spec $@

ldap2pg-el6.spec: Makefile ../setup.py
	cd ..; python setup.py bdist_rpm \
	  --release "1%{dist}" \
	  --build-requires "python2 python2-setuptools" \
	  --requires "python-psycopg2 python-ldap PyYAML python-logutils python-argparse" \
	  --python "/usr/bin/python2" \
	  --spec-only
	cp -f ../dist/ldap2pg.spec $@
