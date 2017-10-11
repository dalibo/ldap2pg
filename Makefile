VERSION=$(shell python setup.py --version)

default:

readme-sample:
	@echo -n '$$ '
	cat ldap2pg.minimal.yml
	@echo -n '$$ '
	ldap2pg --color --config docs/ldap2pg.minimal.yml --real 2>&1 | sed s,${LOGNAME},...,g

release:
	python setup.py egg_info
	sed -i 's/ \+(unreleased)$$//' docs/changelog.md
	git commit setup.py docs/changelog.md -m "Version $(VERSION)"
	git tag $(VERSION)
	@echo
	@echo Now push with
	@echo
	@echo "    git push rw"
	@echo "    git push --tags rw"
	@echo
	@echo and upload with make upload

upload:
	git describe --exact-match --tags
	python3 setup.py sdist bdist_wheel upload -r pypi
