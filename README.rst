=====================================================
 ``ldap2pg`` -- Synchronize Postgres roles from LDAP
=====================================================

| |CircleCI| |Codecov| |RTD| |PyPI|

Swiss-army knife to synchronize Postgres roles from any LDAP directory.

Features
========

- Creates, alter and drops PostgreSQL roles from LDAP queries.
- Creates static roles from YAML to complete LDAP entries.
- Manage role members (alias *groups*).
- Dry run.
- Logs LDAP queries as ``ldapsearch`` commands.
- Logs **every** SQL queries.
- Reads settings from YAML config file.

::

    $ cat ldap2pg.yml
    sync_map:
      ldap:
        base: ou=people,dc=ldap2pg,dc=local
        filter: "(objectClass=organizationalRole)"
        attribute: cn
      role:
        name_attribute: cn
        options: LOGIN
    $ ldap2pg --real
    Using ./ldap2pg.yml.
    Starting ldap2pg 1.0a2.
    Running in real mode.
    Inspecting Postgres...
    Querying LDAP cn=people,dc=ldap2pg,dc=local...
    Create bar.
    Create baz.
    Create bob.
    Create foo.
    Update options of alice.
    Drop spurious.
    Synchronization complete.
    $

See versionned `ldap2pg.yml
<https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml>`_ for further
options.


Installation
============

Install it from PypI tarball::

    pip install ldap2pg


``ldap2pg`` is licensed under `PostgreSQL license
<https://opensource.org/licenses/postgresql>`_.

.. |Codecov| image:: https://codecov.io/gh/dalibo/ldap2pg/branch/master/graph/badge.svg
   :target: https://codecov.io/gh/dalibo/ldap2pg
   :alt: Code coverage report

.. |CircleCI| image:: https://circleci.com/gh/dalibo/ldap2pg.svg?style=shield
   :target: https://circleci.com/gh/dalibo/ldap2pg
   :alt: Continuous Integration report

.. |PyPI| image:: https://img.shields.io/pypi/v/ldap2pg.svg
   :target: https://pypi.python.org/pypi/ldap2pg
   :alt: Version on PyPI

.. |RTD| image:: https://readthedocs.org/projects/ldap2pg/badge/?version=latest
   :target: http://ldap2pg.readthedocs.io/en/latest/?badge=latest
   :alt: Documentation
