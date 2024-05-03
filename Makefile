VERSION=$(shell git describe --tags | grep -Po 'v\K.+')
YUM_LABS?=$(wildcard ../yum-labs)

default:
	@echo ldap2pg $(VERSION)

big: reset-samba1
	while ! bash -c "echo -n > /dev/tcp/$${LDAPURI#*//}/636" ; do sleep 1; done
	test/fixtures/genperfldif.sh | ldapmodify -xw $$LDAPPASSWORD
	$(MAKE) reset-big

reset-big: reset-postgres
	while ! bash -c "echo -n > /dev/tcp/$${PGHOST}/5432" ; do sleep 1; done
	test/fixtures/perf.sh

reset-%:
	docker compose up --force-recreate --no-deps --renew-anon-volumes --detach $*

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

RELEASE_BRANCH=master
RELEASE_REMOTE=git@github.com:dalibo/ldap2pg.git
NEXT_RELEASE:=$(shell grep -m 1 -Po '^# ldap2pg \K.+' CHANGELOG.md)
release:
	git rev-parse --abbrev-ref HEAD | grep -q '^$(RELEASE_BRANCH)$$'
	! grep -iq '^# Unreleased' CHANGELOG.md
	git commit docs/changelog.md -m "New version $(NEXT_RELEASE)"
	git tag v$(NEXT_RELEASE)
	git push $(RELEASE_REMOTE) refs/heads/$(RELEASE_BRANCH):refs/heads/$(RELEASE_BRANCH)
	git push $(RELEASE_REMOTE) tag v$(NEXT_RELEASE)
	@echo Now wait for CI and run make publish-packages;

publish-packages:
	$(MAKE) download-packages
	$(MAKE) publish-deb
	$(MAKE) publish-rpm

CURL=curl --fail --create-dirs --location --silent --show-error
GH_DOWNLOAD=https://github.com/dalibo/ldap2pg/releases/download/v$(VERSION)
PKGBASE=ldap2pg_$(VERSION)_linux_amd64
download-packages:
	$(CURL) --output-dir dist/ --remote-name $(GH_DOWNLOAD)/$(PKGBASE).deb
	$(CURL) --output-dir dist/ --remote-name $(GH_DOWNLOAD)/$(PKGBASE).rpm

dist/$(PKGBASE)_%.changes: dist/$(PKGBASE).deb
	CODENAME=$* build/simplechanges.py $< > $@
	debsign $@

publish-deb:
	rm -vf dist/*.changes
	$(MAKE) dist/$(PKGBASE)_bookworm.changes
	SKIPDEB=1 $(MAKE) dist/$(PKGBASE)_bullseye.changes
	SKIPDEB=1 $(MAKE) dist/$(PKGBASE)_buster.changes
	SKIPDEB=1 $(MAKE) dist/$(PKGBASE)_stretch.changes
	SKIPDEB=1 $(MAKE) dist/$(PKGBASE)_jammy.changes
	@if expr match "$(VERSION)" ".*[a-z]\+" >/dev/null; then echo 'Refusing to publish prerelease $(VERSION) in APT repository.'; false ; fi
	dput labs dist/*.changes

publish-rpm:
	@make -C $(YUM_LABS) clean
	cp dist/$(PKGBASE).rpm $(YUM_LABS)/rpms/RHEL9-x86_64/
	cp dist/$(PKGBASE).rpm $(YUM_LABS)/rpms/RHEL8-x86_64/
	cp dist/$(PKGBASE).rpm $(YUM_LABS)/rpms/RHEL7-x86_64/
	cp dist/$(PKGBASE).rpm $(YUM_LABS)/rpms/RHEL6-x86_64/
	@if expr match "$(VERSION)" ".*[a-z]\+" >/dev/null; then echo 'Refusing to publish prerelease $(VERSION) in YUM repository.'; false ; fi
	@make -C $(YUM_LABS) push createrepos clean

tag-latest:
	docker rmi --force dalibo/ldap2pg:v$(VERSION)
	docker pull dalibo/ldap2pg:v$(VERSION)
	docker tag dalibo/ldap2pg:v$(VERSION) dalibo/ldap2pg:latest
	@if expr match "$(VERSION)" ".*[a-z]\+" >/dev/null; then echo 'Refusing to tag prerelease $(VERSION) as latest in Docker Hub repository.'; false ; fi
	docker push dalibo/ldap2pg:latest
