# Test order matters.

import os


def test_sync(ldap2pg, psql):
    c = 'test/ldap2pg.nonsuper.yml'
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
    myldap2pg('--check', _ok_code=1, _env=env)

    myldap2pg('--real', _env=env)
    roles = list(psql.roles())
    assert 'manuel' in roles
    assert 'kevin' not in roles

    myldap2pg('--check', _env=env)
