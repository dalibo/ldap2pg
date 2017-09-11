# Test order matters.

from __future__ import unicode_literals


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg(c='tests/func/ldap2pg.acl.yml')


def test_check_mode(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--check', c='tests/func/ldap2pg.acl.yml', _ok_code=1)


def test_real_mode(dev, ldap):
    from sh import ldap2pg

    # Ensure database is not synchronized
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml', _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c='tests/func/ldap2pg.acl.yml')
    # Ensure ACL inspects are ok
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')


def test_re_grant(dev, ldap, psql):
    from sh import ldap2pg

    # Ensure db is sync
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')
    # Revoke on one table. This should trigger a re-GRANT
    psql(d=b'frontend', c=b'REVOKE SELECT ON frontend.table2 FROM daniel;')
    # Ensure database is not sync.
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml', _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c='tests/func/ldap2pg.acl.yml')
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')


def test_re_revoke(dev, ldap, psql):
    from sh import ldap2pg

    # Ensure db is sync
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')
    # Partial GRANT to oscar. This must trigger a re-REVOKE
    psql(d=b'frontend', c=b'GRANT SELECT ON frontend.table1 TO oscar;')
    # Ensure database is not sync.
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml', _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c='tests/func/ldap2pg.acl.yml')
    ldap2pg('-C', c='tests/func/ldap2pg.acl.yml')
