VERSION=$(shell cat internal/VERSION)

default:

%.md: %.md.j2 docs/auto-privileges-doc.py ldap2pg/defaults.py Makefile
	echo '<!-- GENERATED FROM $< -->' > $@.tmp
	python docs/auto-privileges-doc.py $< >> $@.tmp
	mv -f $@.tmp $@

.PHONY: docs
docs: docs/wellknown.md
	mkdocs build --clean --strict

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

.PHONY: build
build:
	go build -o build/go-ldap2pg.amd64 -trimpath -buildvcs -ldflags -s ./cmd/go-ldap2pg

release: changelog VERSION
	git commit internal/VERSION docs/changelog.md -m "Version $(VERSION)"
	git tag $(VERSION)
	git push git@github.com:dalibo/ldap2pg.git
	git push --tags git@github.com:dalibo/ldap2pg.git
	@echo Now wait for CI and run make push-rpm;

release-notes:  #: Extract changes for current release
	FINAL_VERSION="$(shell echo $(VERSION) | grep -Po '([^a-z]{3,})')" ; sed -En "/Unreleased/d;/^#+ ldap2pg $$FINAL_VERSION/,/^#/p" CHANGELOG.md  | sed '1d;$$d'

rpm: $(WHL)
	$(MAKE) -C packaging rpm

push-rpm: rpm
	$(MAKE) -C packaging push

reset-%:
	docker-compose up --force-recreate --no-deps --renew-anon-volumes --detach $*
