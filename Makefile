VERSION=$(shell python setup.py --version)

default:

clean-pyc:
	find . -name __pycache__ -or -name "*.pyc" | xargs -rt rm -rf

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
	python setup.py egg_info
	sed -i 's/^# Unreleased$$/# ldap2pg $(VERSION)/' docs/changelog.md

.PHONY: VERSION
VERSION: internal/VERSION
internal/VERSION: setup.py
	echo -n "v$(VERSION).0" > $@

release: changelog VERSION
	git commit internal/VERSION setup.py docs/changelog.md -m "Version $(VERSION)"
	git tag $(VERSION)
	git push git@github.com:dalibo/ldap2pg.git
	git push --tags git@github.com:dalibo/ldap2pg.git
	@echo Now wait for CI and run make push-rpm;

release-notes:  #: Extract changes for current release
	FINAL_VERSION="$(shell echo $(VERSION) | grep -Po '([^a-z]{3,})')" ; sed -En "/Unreleased/d;/^#+ ldap2pg $$FINAL_VERSION/,/^#/p" CHANGELOG.md  | sed '1d;$$d'

WHL=dist/ldap2pg-$(VERSION)-py2.py3-none-any.whl
$(WHL):
	mkdir -p $(dir $@)
	pip3 download --no-deps --dest $(dir $@) ldap2pg==$(VERSION)

rpm: $(WHL)
	$(MAKE) -C packaging rpm

push-rpm: rpm
	$(MAKE) -C packaging push

reset-%:
	docker-compose up --force-recreate --no-deps --renew-anon-volumes --detach $*
