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

    # synchronize all
    ldap2pg('-N', c='tests/func/ldap2pg.acl.yml')
    # Ensure ACL inspects are ok
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')

    # Create a new table.
    psql(d='frontend', c='CREATE TABLE frontend.nt(id INTEGER PRIMARY KEY);')
    # Ensure GRANT ON ALL TABLES IN SCHEMA must be reexecuted.
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml', _ok_code=1)

    # resynchronize all
    ldap2pg('-N', c='tests/func/ldap2pg.acl.yml')
    # Ensure ACL inspects are ok again
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')
