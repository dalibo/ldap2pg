# Test order matters.

from __future__ import unicode_literals


def test_custom_privilege(dev, psql):
    from sh import ldap2pg
    c = 'tests/func/ldap2pg.custom_privilege.yml'

    # Ensure database is not sync.
    ldap2pg('-C', c=c, _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c=c)
    ldap2pg('-C', c=c)
