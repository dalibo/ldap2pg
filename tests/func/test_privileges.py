# Test order matters.

from __future__ import unicode_literals


def test_only_privileges(psql):
    from sh import ldap2pg
    c = 'tests/func/ldap2pg.only_privileges.yml'

    # Ensure database is not sync.
    ldap2pg('-C', c=c, _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c=c)
    ldap2pg('-C', c=c)

    roles = list(psql.roles())

    # Ensure o* role is not dropped.
    assert 'oscar' in roles

    assert 'f' == psql.scalar(
        "SELECT has_language_privilege('public', 'plpgsql', 'USAGE');"
    )
