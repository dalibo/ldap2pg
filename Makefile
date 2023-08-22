VERSION=$(shell cat internal/VERSION)
YUM_LABS?=$(wildcard ../yum-labs)

default:

%.md: %.md.j2 docs/auto-privileges-doc.py ldap2pg/defaults.py Makefile
	echo '<!-- GENERATED FROM $< -->' > $@.tmp
	python docs/auto-privileges-doc.py $< >> $@.tmp
	mv -f $@.tmp $@

.PHONY: docs
docs: docs/wellknown.md
	mkdocs build --clean --strict

build-docker:
	docker build --build-arg http_proxy -t dalibo/ldap2pg:local -f docker/Dockerfile .

big: reset-ldap
	while ! bash -c "echo -n > /dev/tcp/$${LDAPURI#*//}/636" ; do sleep 1; done
	test/fixtures/genperfldif.sh | ldapmodify -xw integral
	$(MAKE) reset-big

reset-big: reset-postgres
	while ! bash -c "echo -n > /dev/tcp/$${PGHOST}/5432" ; do sleep 1; done
	test/fixtures/perf.sh

readme-sample:
	@ldap2pg --config docs/readme/ldap2pg.yml --real
	@psql -f docs/readme/reset.sql
	@echo '$$ cat ldap2pg.yml'
	@cat docs/readme/ldap2pg.yml
	@echo '$$ ldap2pg --real'
	@ldap2pg --color --config docs/readme/ldap2pg.yml --real 2>&1 | sed s,${PWD}/docs/readme,...,g
	@echo '$$ '
	@echo -e '\n\n\n\n'

changelog:
	sed -i 's/^# Unreleased$$/# ldap2pg $(VERSION)/' docs/changelog.md

release: changelog
	git commit internal/VERSION docs/changelog.md -m "Version $(VERSION)"
	git tag $(VERSION)
	git push --follow-tags git@github.com:dalibo/ldap2pg.git refs/heads/master:refs/heads/master
	@echo Now wait for CI and run make push-rpm;

rpm deb:
	VERSION=$(VERSION) nfpm package --packager $@

publish-rpm: rpm
	cp build/ldap2pg-$(VERSION).x86_64.rpm $(YUM_LABS)/rpms/RHEL8-x86_64/
	cp build/ldap2pg-$(VERSION).x86_64.rpm $(YUM_LABS)/rpms/RHEL7-x86_64/
	cp build/ldap2pg-$(VERSION).x86_64.rpm $(YUM_LABS)/rpms/RHEL6-x86_64/
	@make -C $(YUM_LABS) push createrepos clean

reset-%:
	docker-compose up --force-recreate --no-deps --renew-anon-volumes --detach $*
