# Test order matters.

from __future__ import unicode_literals

import pytest


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--verbose')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'oscar' in roles
    assert 'alice' in superusers


def test_real_mode(dev, ldap, psql):
    from sh import ErrorReturnCode, ldap2pg

    assert 'keepme' in psql.tables(dbname='legacy')
    # Assert daniel can connect to backend
    psql(U='daniel', d='backend', c='SELECT CURRENT_USER')

    print(ldap2pg('-vN', c='ldap2pg.master.yml'))

    roles = list(psql.roles())
    assert 'alan' in roles
    assert 'oscar' not in roles

    assert 'alice' in psql.superusers()

    assert 'dave' in psql.members('backend')
    assert 'david' in psql.members('frontend')
    assert 'alice' in psql.members('ldap_users')

    # Assert that table keepme owned by deleted user spurious is not dropped!
    assert 'keepme' in psql.tables(dbname='legacy')

    # Assert CONNECT to backend has been revoked from daniel.
    with pytest.raises(ErrorReturnCode):
        psql(U='daniel', d='backend', c='SELECT CURRENT_USER')


def test_nothing_to_do():
    from sh import ldap2pg

    out = ldap2pg('--real', '--config', 'ldap2pg.master.yml')

    assert b'Nothing to do' in out.stderr
