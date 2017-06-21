=====================================================
 ``ldap2pg`` -- Synchronize Postgres roles from LDAP
=====================================================

| |CircleCI| |Codecov|


Features
========

- Creates and drops PostgreSQL roles from LDAP query
- Reads settings from YAML config file

::

    $ ldap2pg
     INFO Starting ldap2pg 0.1.
     INFO Creating new role alice.
    WARNI Dropping existing role toto.
     INFO Synchronization complete.
    $


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
