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
# building rpm update egg_info for python2.7. We need to regenerate it for
# development.
	cd ../ && python setup.py egg_info

rpm-centos%:
	DIST=centos$* docker-compose run --rm rpm
	cp -fl $$(readlink -e $(TOPSRC_DIR)/dist/ldap2pg-last.rpm) $(YUM_LABS)/rpms/CentOS$*-x86_64/

clean:
	docker-compose down -v

push:
	@make -C $(YUM_LABS) push createrepos clean
