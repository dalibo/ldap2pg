VERSION=$(shell grep -Po 'v\K.+' internal/VERSION)
YUM_LABS?=$(wildcard ../yum-labs)

default:
	@echo ldap2pg $(VERSION)

big: reset-ldap
	while ! bash -c "echo -n > /dev/tcp/$${LDAPURI#*//}/636" ; do sleep 1; done
	test/fixtures/genperfldif.sh | ldapmodify -xw integral
	$(MAKE) reset-big

reset-big: reset-postgres
	while ! bash -c "echo -n > /dev/tcp/$${PGHOST}/5432" ; do sleep 1; done
	test/fixtures/perf.sh

reset-%:
	docker-compose up --force-recreate --no-deps --renew-anon-volumes --detach $*

readme-sample:
	@test/ldap2pg.sh --config docs/readme/ldap2pg.yml --real
	@psql -f docs/readme/reset.sql
	@echo '$$ cat ldap2pg.yml'
	@cat docs/readme/ldap2pg.yml
	@echo '$$ ldap2pg --real'
	@test/ldap2pg.sh --color --config docs/readme/ldap2pg.yml --real 2>&1 | sed s,${PWD}/docs/readme,...,g
	@echo '$$ '
	@echo -e '\n\n\n\n'

%.md: %.md.tmpl cmd/render-doc/main.go Makefile
	echo '<!-- GENERATED FROM $< FOR v$(VERSION) -->' > $@.tmp
	go run ./cmd/render-doc $< >> $@.tmp
	mv -f $@.tmp $@

.PHONY: docs
docs: docs/builtins.md
	mkdocs build --clean --strict

build-docker:
	docker build --build-arg http_proxy -t dalibo/ldap2pg:local -f docker/Dockerfile .

release: changelog
	sed -i 's/^# Unreleased$$/# ldap2pg $(VERSION)/' docs/changelog.md
	git commit internal/VERSION docs/changelog.md -m "Version $(VERSION)"
	git tag v$(VERSION)
	git push --follow-tags git@github.com:dalibo/ldap2pg.git refs/heads/master:refs/heads/master
	@echo Now wait for CI and run make push-rpm;

CURL=curl --fail --create-dirs --location --silent --show-error
GH_DOWNLOAD=https://github.com/dalibo/ldap2pg/releases/download/v$(VERSION)
PKGBASE=ldap2pg_$(VERSION)_linux_amd64
download-packages:
	$(CURL) --output-dir dist/ --remote-name $(GH_DOWNLOAD)/$(PKGBASE).deb
	$(CURL) --output-dir dist/ --remote-name $(GH_DOWNLOAD)/$(PKGBASE).rpm

publish-rpm:
	cp dist/ldap2pg-$(VERSION).x86_64.rpm $(YUM_LABS)/rpms/RHEL8-x86_64/ $(YUM_LABS)/rpms/RHEL7-x86_64/ $(YUM_LABS)/rpms/RHEL6-x86_64/
	@make -C $(YUM_LABS) push createrepos clean
