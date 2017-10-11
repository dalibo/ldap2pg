|ldap2pg|

| |CircleCI| |Codecov| |Quality| |RTD| |PyPI|

Swiss-army knife to synchronize Postgres roles and ACLs from any LDAP directory.

.. _documentation: https://ldap2pg.readthedocs.io/en/latest/
.. _license:       https://opensource.org/licenses/postgresql


Features
========

- Creates, alter and drops PostgreSQL roles from LDAP queries.
- Creates static roles from YAML to complete LDAP entries.
- Manage role members (alias *groups*).
- Grant or revoke custom ACL statically or from LDAP entries.
- Dry run.
- Logs LDAP queries as ``ldapsearch`` commands.
- Logs **every** SQL queries.
- Reads settings from YAML config file.

Here is a sample configuration and execution:

::

    $ cat ldap2pg.minimal.yml
    sync_map:
    - role:
        name: ldap
        options: NOLOGIN
    - ldap:
        base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
        filter: "(objectClass=organizationalRole)"
        attribute: cn
      role:
        name_attribute: cn
        options: LOGIN
        parent: ldap
    $ ldap2pg --color --config ldap2pg.minimal.yml --real 2>&1 | sed s,bersace,...,g
    Starting ldap2pg 2.0a3.
    Using /home/.../src/dalibo/ldap2pg/ldap2pg.minimal.yml.
    Running in real mode.
    Inspecting Postgres...
    Querying LDAP ou=people,dc=ldap,dc=ldap2pg,dc=docker...
    Create alan.
    Create albert.
    Create dave.
    Create donald.
    Create ldap.
    Add ldap members.
    Update options of alice.
    Reassign oscar objects and purge ACL on frontend.
    Reassign oscar objects and purge ACL on postgres.
    Reassign oscar objects and purge ACL on template1.
    Drop oscar.
    Synchronization complete.
    $

See versionned `ldap2pg.yml
<https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml>`_ and documentation_
for further options.


Installation
============

Install it from PyPI tarball::

    pip install ldap2pg

More details can be found in documentation_.


``ldap2pg`` is licensed under PostgreSQL license_.


.. |Codecov| image:: https://codecov.io/gh/dalibo/ldap2pg/branch/master/graph/badge.svg
   :target: https://codecov.io/gh/dalibo/ldap2pg
   :alt: Code coverage report

.. |CircleCI| image:: https://circleci.com/gh/dalibo/ldap2pg.svg?style=shield
   :target: https://circleci.com/gh/dalibo/ldap2pg
   :alt: Continuous Integration report

.. |Quality| image:: https://landscape.io/github/dalibo/ldap2pg/master/landscape.svg?style=flat
   :target: https://landscape.io/github/dalibo/ldap2pg/master
   :alt: Code Health

.. |ldap2pg| image:: https://github.com/dalibo/ldap2pg/raw/master/docs/img/logo-phrase.png
   :target: https://github.com/dalibo/ldap2pg
   :alt: ldap2pg: PostgreSQL role and ACL management

.. |PyPI| image:: https://img.shields.io/pypi/v/ldap2pg.svg
   :target: https://pypi.python.org/pypi/ldap2pg
   :alt: Version on PyPI

.. |RTD| image:: https://readthedocs.org/projects/ldap2pg/badge/?version=latest
   :target: https://ldap2pg.readthedocs.io/en/latest/?badge=latest
   :alt: Documentation
