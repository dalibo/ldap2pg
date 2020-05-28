# Test order matters.

from __future__ import unicode_literals


def test_run(psql):
    from sh import ldap2pg
    c = 'tests/func/ldap2pg.full.yml'

    # Ensure database is not sync.
    ldap2pg('-C', c=c, _ok_code=1)

    # Synchronize all
    ldap2pg('-N', c=c)
    ldap2pg('-C', c=c)

    roles = list(psql.roles())

    assert 'Alan' in roles
    assert 'oscar' not in roles

    assert 'ALICE' in psql.superusers()

    writers = list(psql.members('writers'))

    assert 'daniel' in writers
    assert 'david' in writers
    assert 'didier' in writers
    assert 'ALICE' in psql.members('ldap_roles')

    comment = psql.scalar("""\
    SELECT description
    FROM pg_shdescription
    WHERE description = 'mail: alice@ldap2pg.docker';
    """)
    assert comment
