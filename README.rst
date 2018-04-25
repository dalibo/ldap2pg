|ldap2pg|

| |CircleCI| |Codecov| |RTD| |PyPI|

Swiss-army knife to synchronize Postgres roles and privileges from YAML or LDAP.

.. _documentation: https://ldap2pg.readthedocs.io/en/latest/
.. _license:       https://opensource.org/licenses/postgresql
.. _contributors:  https://github.com/dalibo/ldap2pg/blob/master/CONTRIBUTING.md#contributors


Features
========

- Creates, alter and drops PostgreSQL roles from LDAP queries.
- Creates static roles from YAML to complete LDAP entries.
- Manage role members (alias *groups*).
- Grant or revoke privileges statically or from LDAP entries.
- Dry run.
- Logs LDAP queries as ``ldapsearch`` commands.
- Logs **every** SQL queries.
- Reads settings from an expressive YAML config file.

Here is a sample configuration and execution:

::

    $ cat docs/ldap2pg.minimal.yml
    - role:
        name: ldap
        options: NOLOGIN
    - ldap:
        base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
        filter: "(objectClass=organizationalRole)"
      role:
        name_attribute: cn
        options: LOGIN
        parent: ldap
    $ ldap2pg --config docs/ldap2pg.minimal.yml --real
    Starting ldap2pg 4.5.
    Using .../docs/ldap2pg.minimal.yml.
    Running in real mode.
    Inspecting Postgres roles...
    Querying LDAP ou=people,dc=ldap,dc=lda... (objectClass...
    Create albert.
    Create alter.
    Create didier.
    Create dorothée.
    Create ldap.
    Add ldap members.
    Update options of alice.
    Update options of alan.
    Reassign olivier objects and purge ACL on appdb.
    Reassign olivier objects and purge ACL on olddb.
    Drop olivier.
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


``ldap2pg`` is licensed under PostgreSQL license_. ``ldap2pg`` is available with
the help of wonderful people, jump to contributors_ list to see them.


.. |Codecov| image:: https://codecov.io/gh/dalibo/ldap2pg/branch/master/graph/badge.svg
   :target: https://codecov.io/gh/dalibo/ldap2pg
   :alt: Code coverage report

.. |CircleCI| image:: https://circleci.com/gh/dalibo/ldap2pg.svg?style=shield
   :target: https://circleci.com/gh/dalibo/ldap2pg
   :alt: Continuous Integration report

.. |ldap2pg| image:: https://github.com/dalibo/ldap2pg/raw/master/docs/img/logo-phrase.png
   :target: https://github.com/dalibo/ldap2pg
   :alt: ldap2pg: PostgreSQL role and privileges management

.. |PyPI| image:: https://img.shields.io/pypi/v/ldap2pg.svg
   :target: https://pypi.python.org/pypi/ldap2pg
   :alt: Version on PyPI

.. |RTD| image:: https://readthedocs.org/projects/ldap2pg/badge/?version=latest
   :target: https://ldap2pg.readthedocs.io/en/latest/?badge=latest
   :alt: Documentation
