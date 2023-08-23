import pytest


@pytest.fixture(scope='module')
def nominalrun(ldap2pg):
    ldap2pg = ldap2pg.bake(c='ldap2pg.yml')

    # Ensure database is not sync.
    ldap2pg('--check', _ok_code=1)

    # Synchronize all
    ldap2pg('--real')
    ldap2pg('--check')
    return ldap2pg


def test_roles(nominalrun, psql):
    roles = list(psql.roles())

    assert 'alain' in roles
    assert 'daniel' not in roles

    readers = list(psql.members('readers'))
    assert 'corinne' in readers

    writers = list(psql.members('writers'))
    assert 'alice' in writers

    owners = list(psql.members('owners'))
    assert 'alter' in owners
    assert 'alain' not in owners


def test_re_grant(nominalrun, psql):
    psql(c='REVOKE CONNECT ON DATABASE nominal FROM readers;')
    ldap2pg = nominalrun
    # Ensure database is not sync.
    ldap2pg('--check', _ok_code=1)
    # Synchronize all
    ldap2pg('--real')
    ldap2pg('--check')


def test_re_revoke(nominalrun, psql):
    ldap2pg = nominalrun

    psql(c='GRANT CREATE ON SCHEMA public TO readers;')
    # Ensure database is not sync.
    ldap2pg('--check', _ok_code=1)
    # Synchronize all
    ldap2pg('--real')
    ldap2pg('--check')


def test_nothing_to_do(nominalrun, capsys):
    nominalrun('--real', '--check')

    _, err = capsys.readouterr()
    assert 'Nothing to do' in err
