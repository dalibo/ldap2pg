# Test order matters.

from __future__ import unicode_literals

import os


def test_non_superuser(dev, psql):
    from sh import ldap2pg
    c = 'tests/func/ldap2pg.nonsuper.yml'
    db = 'nonsuperdb'
    env = dict(
        os.environ,
        PGUSER='nonsuper',
        PGDATABASE=db,
    )
    env.pop('PGDSN', None)
    myldap2pg = ldap2pg.bake(c=c, _env=env)

    # Create a table owned by manager

    # Ensure db is not sync
    myldap2pg('-C', _ok_code=1, _env=env)

    myldap2pg('-N', _env=env)
    roles = list(psql.roles())
    assert 'manuel' in roles
    assert 'kevin' not in roles

    myldap2pg('-C', _env=env)


def test_dry_run(dev, psql):
    from sh import ldap2pg

    ldap2pg('--verbose', config='ldap2pg.yml')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    # oscar is not dropped
    assert 'oscar' in roles
    assert 'alice' in superusers


def test_check_mode(dev, psql):
    from sh import ldap2pg

    ldap2pg('--check', config='ldap2pg.yml', _ok_code=1)


def test_real_mode(dev, psql):
    from sh import ldap2pg

    assert 'keepme' in psql.tables(dbname='olddb')

    ldap2pg('-N', c='ldap2pg.yml')
    # Workaround bug in Postgres: execute on functions to public persists
    # revoke.
    ldap2pg('-N', c='ldap2pg.yml')

    roles = list(psql.roles())
    writers = list(psql.members('writers'))

    assert 'alan' in roles
    assert 'oscar' not in roles

    assert 'alice' in psql.superusers()

    assert 'daniel' in writers
    assert 'david' in writers
    assert 'didier' in writers
    assert 'alice' in psql.members('ldap_roles')

    # Assert that table keepme owned by deleted user spurious is not dropped!
    assert 'keepme' in psql.tables(dbname='olddb')
    assert 'keepme' in roles


def test_nothing_to_do(capsys, dev):
    from sh import ldap2pg

    ldap2pg('--real', config='ldap2pg.yml')

    _, err = capsys.readouterr()
    assert 'Nothing to do' in err


def test_re_grant(dev, psql):
    from sh import ldap2pg

    # Ensure db is sync
    ldap2pg('-C', c='ldap2pg.yml')
    # Revoke on one table. This must trigger a re-GRANT
    psql(d=b'appdb', c=b'REVOKE SELECT ON appns.table2 FROM readers;')
    # Ensure database is not sync.
    ldap2pg('-C', c='ldap2pg.yml', _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c='ldap2pg.yml')
    ldap2pg('-C', c='ldap2pg.yml')


def test_re_revoke(dev, psql):
    from sh import ldap2pg
    c = 'ldap2pg.yml'

    # Ensure db is sync
    ldap2pg('-C', c=c)
    # Partial GRANT to oscar. This must trigger a re-REVOKE
    psql(d=b'appdb', c=b'GRANT INSERT ON appns.table1 TO readers;')
    # Ensure database is not sync.
    ldap2pg('-C', c=c, _ok_code=1)
    # Synchronize all
    ldap2pg('-N', c=c)
    ldap2pg('-C', c=c)


def test_joins_in_real_mode(dev, psql):
    from sh import ldap2pg

    ldap2pg('-N', c='tests/func/ldap2pg.joins.yml')

    roles = list(psql.roles())
    writers = list(psql.members('writers'))

    assert 'alan@ldap2pg.docker' in roles
    assert 'oscar@ldap2pg.docker' not in roles

    assert 'alice@ldap2pg.docker' in psql.superusers()

    assert 'daniel@ldap2pg.docker' in writers
    assert 'david@ldap2pg.docker' not in writers
    assert 'didier@ldap2pg.docker' in writers
    assert 'alice@ldap2pg.docker' in psql.members('ldap_roles')
