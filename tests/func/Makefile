.ONESHELL:
default:

rpm:
	make -C ../../packaging rpm

tests:
	touch bash_history
	docker-compose up --abort-on-container-exit runner

clean:
	make -C ../../packaging clean
	docker-compose down -v

debug:
	docker-compose exec runner /bin/bash

pytest:
	cd ../..
	find . -name "*.pyc" -delete
# Looks like ./ldaprc is ignored in CentOS7 by libldap and tools. Put it in envs.
	env $(shell sed 's/^/LDAP/;s/ \+/=/g' ../../ldaprc) pytest tests/func/ $(filter-out $@,$(MAKECMDGOALS))

%:
	@echo Passing $@ as argument.
