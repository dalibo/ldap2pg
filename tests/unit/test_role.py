from __future__ import unicode_literals

from fnmatch import filter as fnfilter

import pytest


def test_role():
    from ldap2pg.manager import Role

    role = Role(name='toto')

    assert 'toto' == role.name
    assert 'toto' == str(role)
    assert 'toto' in repr(role)


def test_options():
    from ldap2pg.manager import RoleOptions

    options = RoleOptions()

    assert 'NOSUPERUSER' in repr(options)

    with pytest.raises(ValueError):
        options.update(dict(POUET=True))

    with pytest.raises(ValueError):
        RoleOptions(POUET=True)


def test_roles_diff_queries():
    from ldap2pg.manager import Role, RoleSet

    a = RoleSet([
        Role('drop-me'),
        Role('alter-me'),
        Role('nothing'),
    ])
    b = RoleSet([
        Role('alter-me', options=dict(LOGIN=True)),
        Role('nothing'),
        Role('create-me')
    ])
    queries = list(a.diff(b))

    assert fnfilter(queries, "ALTER ROLE alter-me WITH* LOGIN*;")
    assert fnfilter(queries, "CREATE ROLE create-me *;")
    assert 'DROP ROLE drop-me;' in queries
    assert not fnfilter(queries, '*nothing*')
