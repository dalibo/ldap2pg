# Test order matters.

from __future__ import unicode_literals


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--verbose', config='ldap2pg.yml')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'oscar' in roles
    assert 'alice' in superusers


def test_check_mode(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--check', config='ldap2pg.yml', _ok_code=1)


def test_real_mode(dev, ldap, psql):
    from sh import ldap2pg

    assert 'keepme' in psql.tables(dbname='olddb')

    ldap2pg('-vN', c='ldap2pg.yml')

    roles = list(psql.roles())
    appmembers = list(psql.members('app'))

    assert 'alan' in roles
    assert 'oscar' not in roles

    assert 'alice' in psql.superusers()

    assert 'daniel' in appmembers
    assert 'david' in appmembers
    assert 'alice' in psql.members('ldap_users')

    # Assert that table keepme owned by deleted user spurious is not dropped!
    assert 'keepme' in psql.tables(dbname='olddb')


def test_nothing_to_do():
    from sh import ldap2pg

    out = ldap2pg('--real', config='ldap2pg.yml')

    assert b'Nothing to do' in out.stderr


def test_custom_query(psql):
    from sh import ldap2pg

    # Ensure we have a role not matching `d%`
    roles = list(psql.roles())
    assert 'alan' in roles

    yaml = open('tests/func/ldap2pg.custom_inspect.yml')
    out = ldap2pg('-v', '--config=-', _in=yaml)

    # However, alan is not dopped.
    assert b'Nothing to do' in out.stderr
