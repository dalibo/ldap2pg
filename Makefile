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
	@echo -n '$$ '
	cat docs/ldap2pg.minimal.yml
	@echo '$$ ldap2pg --config docs/ldap2pg.minimal.yml --real'
	@ldap2pg --config docs/ldap2pg.minimal.yml --real 2>&1 | sed s,${PWD},...,g

changelog:
	python setup.py egg_info
	sed -i 's/^# Unreleased$$/# ldap2pg $(VERSION)/' docs/changelog.md

release: changelog
	git commit setup.py docs/changelog.md -m "Version $(VERSION)"
	git tag $(VERSION)
	git push git@github.com:dalibo/ldap2pg.git
	git push --tags git@github.com:dalibo/ldap2pg.git
	@echo Now upload with make upload

upload:
	git describe --exact-match --tags
	python3 setup.py sdist bdist_wheel upload -r pypi

reset-%:
	docker-compose up --force-recreate --no-deps --renew-anon-volumes --detach $*
