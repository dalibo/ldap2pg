=======================================================
 ``ldap2pg`` -- Synchronize PostgresQL roles from LDAP
=======================================================

| |CircleCI| |Codecov|


Features
========

- Create and drop PostgreSQL roles from LDAP query
- Reads settings from YAML config file

::

    $ ldap2pg
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
