# Test order matters.

from __future__ import unicode_literals


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg(c='tests/func/ldap2pg.acl.yml')


def test_check_mode(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--check', c='tests/func/ldap2pg.acl.yml', _ok_code=1)


def test_real_mode(dev, ldap, psql):
    from sh import ldap2pg

    # Ensure GRANT ON ALL TABLES IN SCHEMA must be reexecuted.
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml', _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c='tests/func/ldap2pg.acl.yml')
    # Ensure ACL inspects are ok
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')
