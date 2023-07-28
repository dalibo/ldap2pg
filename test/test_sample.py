# Test order matters.

from __future__ import unicode_literals


def test_dry_run(ldap2pg, psql):
    ldap2pg('--verbose', config='ldap2pg.yml')
    roles = list(psql.roles())
    # daniel is not dropped
    assert 'daniel' in roles


def test_check_mode(ldap2pg, psql):
    ldap2pg('--check', config='ldap2pg.yml', _ok_code=1)


def test_real_mode(ldap2pg, psql):
    ldap2pg('--real', c='ldap2pg.yml')
    # Workaround bug in Postgres: execute on functions to public persists
    # revoke.
    ldap2pg('--real', c='ldap2pg.yml')

    roles = list(psql.roles())

    assert 'alain' in roles
    assert 'daniel' not in roles

    readers = list(psql.members('readers'))
    assert 'corinne' in readers

    writers = list(psql.members('writers'))
    assert 'alice' in writers

    owners = list(psql.members('owners'))
    assert 'alter' in owners


def test_re_grant(ldap2pg, psql):
    # Ensure db is sync
    ldap2pg('--check', c='ldap2pg.yml')
    psql(c='REVOKE CONNECT ON DATABASE nominal FROM readers;')
    # Ensure database is not sync.
    ldap2pg('--check', c='ldap2pg.yml', _ok_code=1)
    # Synchronize all
    ldap2pg('--real', c='ldap2pg.yml')
    ldap2pg('--check', c='ldap2pg.yml')


def test_re_revoke(ldap2pg, psql):
    c = 'ldap2pg.yml'

    # Ensure db is sync
    ldap2pg('--check', c=c)
    psql(c='GRANT CREATE ON SCHEMA public TO readers;')
    # Ensure database is not sync.
    ldap2pg('--check', c=c, _ok_code=1)
    # Synchronize all
    ldap2pg('--real', c=c)
    ldap2pg('--check', c=c)


def test_nothing_to_do(ldap2pg, capsys):
    ldap2pg('--real', config='ldap2pg.yml')

    _, err = capsys.readouterr()
    assert 'Nothing to do' in err
