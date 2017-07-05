# Test order matters.

from __future__ import unicode_literals


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--verbose')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'spurious' in roles
    assert 'alice' in superusers


def test_real_mode(dev, ldap, psql):
    from sh import ldap2pg

    assert 'keepme' in psql.tables(dbname='app0')

    ldap2pg('-vN')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'bob' in roles
    assert 'spurious' not in roles
    assert 'alice' in superusers

    assert 'foo' in psql.members('app0')
    assert 'bar' in psql.members('app1')
    assert 'alice' in psql.members('ldap_users')

    # Assert that table keepme owned by deleted user spurious is not dropped!
    assert 'keepme' in psql.tables(dbname='app0')


def test_nothing_to_do():
    from sh import ldap2pg

    out = ldap2pg('--real')

    assert b'Nothing to do' in out.stderr
