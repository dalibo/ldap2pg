# Functionnal tests

Functionnal tests tends to integrate ldap2pg in real world: execution in target
system, installation from rpm package, no mocks.

Run `make clean rpm tests` to recreate rpm and test env.

On error, the container wait forever. Either enter the container with `make
debug` or kill it with ^C. To reduce dev loop, just `pip install -e .` to use
wip code rather than rpm version.

Tests are written with the great [pytest](https://doc.pytest.org)
and [sh](https://amoffat.github.io/sh/) projects. `conftest.py` provides various
specific fixtures. The most important is that Postgres database and OpenLDAP
base is purged between each modules (not each tests!). Also beware that func
tests are executed in definition order. This allow to split a big scenario in
severals steps without loosing context and CPU cycle.

Two main fixtures are very useful when testing: `psql` and `ldap`. These little
helpers provide fastpath to recurrent inspection of Postgres database on LDAP
base with `sh.py`-style API.
