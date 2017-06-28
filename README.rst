=====================================================
 ``ldap2pg`` -- Synchronize Postgres roles from LDAP
=====================================================

| |CircleCI| |Codecov|

Swiss-army knife to synchronize Postgres roles from any LDAP directory.

Features
========

- Creates and drops PostgreSQL roles from LDAP queries.
- Manage role options (``CREATE`` and ``ALTER``).
- Manage role members (alias *groups*).
- Dry run.
- logs LDAP queries as ``ldapsearch`` commands.
- logs **every** SQL queries.
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
    $ ldap2pg
     INFO Starting ldap2pg 0.1.
     INFO Creating new role alice.
    WARNI Dropping existing role toto.
     INFO Synchronization complete.
    $

See versionned `ldap2pg.yml
<https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml>`_ for further
options.


Installation
============

Install it from GitHub tarball::

    pip install https://github.com/dalibo/ldap2pg/archive/master.zip


``ldap2pg`` is licensed under `PostgreSQL license
<https://opensource.org/licenses/postgresql>`_.

.. |Codecov| image:: https://codecov.io/gh/dalibo/ldap2pg/branch/master/graph/badge.svg
   :target: https://codecov.io/gh/dalibo/ldap2pg
   :alt: Code coverage report

.. |CircleCI| image:: https://circleci.com/gh/dalibo/ldap2pg.svg?style=svg
   :target: https://circleci.com/gh/dalibo/ldap2pg
   :alt: Continuous Integration report
