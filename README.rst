|ldap2pg|

| |CircleCI| |Codecov| |RTD| |PyPI| |Docker|

Swiss-army knife to synchronize Postgres roles and privileges from YAML or LDAP.

.. _documentation: https://ldap2pg.readthedocs.io/en/latest/
.. _license:       https://opensource.org/licenses/postgresql
.. _contributors:  https://github.com/dalibo/ldap2pg/blob/master/CONTRIBUTING.md#contributors


Features
========

- Creates, alters and drops PostgreSQL roles from LDAP queries.
- Creates static roles from YAML to complete LDAP entries.
- Manages role members (alias *groups*).
- Grants or revokes privileges statically or from LDAP entries.
- Dry run.
- Logs LDAP queries as ``ldapsearch`` commands.
- Logs **every** SQL query.
- Reads settings from an expressive YAML config file.

Here is a sample configuration and execution:

::

    $ cat ldap2pg.yml
    - role:
        name: ldap_roles
        options: NOLOGIN
    - ldap:
        base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
        filter: "(objectClass=organizationalPerson)"
      role:
        name: '{cn}'
        options: LOGIN
        parent: ldap_roles
    $ ldap2pg --real
    Starting ldap2pg 5.0.
    Using .../ldap2pg.yml.
    Running in real mode.
    Inspecting roles in Postgres cluster...
    Querying LDAP ou=people,dc=ldap,dc=lda... (objectClass...
    Create domitille.
    Update options of albert.
    Add missing ldap_roles members.
    Delete spurious ldap_roles members.
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

    pip install ldap2pg psycopg2-binary

More details can be found in documentation_.


``ldap2pg`` is licensed under PostgreSQL license_. ``ldap2pg`` is available with
the help of wonderful people, jump to contributors_ list to see them.


Support
=======

If you need support and you didn't found it in documentation_, just drop a
question in a `GitHub issue <https://github.com/dalibo/ldap2pg/issues/new>`_!
French accepted. Don't miss the `cookbook
<https://ldap2pg.readthedocs.io/en/latest/cookbook/>`_. You're welcome!


.. |Codecov| image:: https://codecov.io/gh/dalibo/ldap2pg/branch/master/graph/badge.svg
   :target: https://codecov.io/gh/dalibo/ldap2pg
   :alt: Code coverage report

.. |CircleCI| image:: https://circleci.com/gh/dalibo/ldap2pg.svg?style=shield
   :target: https://circleci.com/gh/dalibo/ldap2pg
   :alt: Continuous Integration report

.. |Docker| image:: https://img.shields.io/docker/automated/dalibo/ldap2pg.svg
   :target: https://hub.docker.com/r/dalibo/ldap2pg
   :alt: Docker Image Available

.. |ldap2pg| image:: https://github.com/dalibo/ldap2pg/raw/master/docs/img/logo-phrase.png
   :target: https://labs.dalibo.com/ldap2pg
   :alt: ldap2pg: PostgreSQL role and privileges management

.. |PyPI| image:: https://img.shields.io/pypi/v/ldap2pg.svg
   :target: https://pypi.python.org/pypi/ldap2pg
   :alt: Version on PyPI

.. |RTD| image:: https://readthedocs.org/projects/ldap2pg/badge/?version=latest
   :target: https://ldap2pg.readthedocs.io/en/latest/?badge=latest
   :alt: Documentation
