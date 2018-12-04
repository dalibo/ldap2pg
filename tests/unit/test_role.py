from __future__ import unicode_literals

from fnmatch import filter as fnfilter

import pytest


def test_role():
    from ldap2pg.manager import Role

    role = Role(name='toto')

    assert 'toto' == role.name
    assert 'toto' == str(role)
    assert 'toto' in repr(role)

    roles = sorted([Role('b'), Role('a')])

    assert ['a', 'b'] == roles


def test_create():
    from ldap2pg.manager import Role

    role = Role(name='toto', members=['titi'], comment='Mycom')

    queries = [q.args[0] for q in role.create()]

    assert fnfilter(queries, 'CREATE ROLE "toto" *;')
    assert fnfilter(queries, 'GRANT "toto" TO "titi";')
    assert fnfilter(queries, '*Mycom*')


def test_alter():
    from ldap2pg.manager import Role

    a = Role(name='toto', members=['titi'], options=dict(LOGIN=True))
    b = Role(name='toto', members=['tata'], options=dict(LOGIN=False))

    queries = [q.args[0] for q in a.alter(a)]
    assert not queries

    queries = [q.args[0] for q in a.alter(b)]

    assert fnfilter(queries, 'ALTER ROLE "toto" *;')
    assert fnfilter(queries, 'GRANT "toto" TO "tata";')
    assert fnfilter(queries, 'REVOKE "toto" FROM "titi";')


def test_drop():
    from ldap2pg.manager import Role

    role = Role(name='toto', members=['titi'])

    queries = [q.args[0] for q in role.drop()]

    assert fnfilter(queries, '*REASSIGN OWNED*DROP OWNED BY "toto";*')
    assert fnfilter(queries, 'DROP ROLE "toto";')


def test_merge():
    from ldap2pg.role import Role

    a = Role(name='daniel', parents=['group0'])
    b = Role(name='daniel', parents=['group1'])
    c = Role(name='daniel', members=['group2'])

    a.merge(b)
    assert 2 == len(a.parents)

    a.merge(c)
    assert 1 == len(a.members)


def test_merge_options():
    from ldap2pg.role import Role

    a = Role(name='daniel', options=dict(SUPERUSER=True))
    b = Role(name='daniel', options=dict(SUPERUSER=False))
    c = Role(name='daniel', options=dict(SUPERUSER=None))
    d = Role(name='daniel', options=dict(SUPERUSER=None))

    with pytest.raises(ValueError):
        a.merge(b)

    # True is kept
    a.merge(c)
    assert a.options['SUPERUSER'] is True
    # False is kept
    b.merge(c)
    assert b.options['SUPERUSER'] is False
    # None is kept
    c.merge(d)
    assert c.options['SUPERUSER'] is None
    # None is replaced.
    d.merge(a)
    assert d.options['SUPERUSER'] is True


def test_options():
    from ldap2pg.role import RoleOptions

    options = RoleOptions()
    options.fill_with_defaults()

    assert 'NOSUPERUSER' in repr(options)

    with pytest.raises(ValueError):
        options.update(dict(POUET=True))

    with pytest.raises(ValueError):
        RoleOptions(POUET=True)


def test_flatten():
    from ldap2pg.role import RoleSet, Role

    roles = RoleSet()
    roles.add(Role('parent0', members=['child0', 'child1']))
    roles.add(Role('parent1', members=['child2', 'child3']))
    roles.add(Role('parent2', members=['child4']))
    roles.add(Role('child0', members=['subchild0']))
    roles.add(Role('child1', members=['subchild1', 'subchild2']))
    roles.add(Role('child2', members=['outer0']))
    roles.add(Role('child3'))
    roles.add(Role('child4'))
    roles.add(Role('subchild0'))
    roles.add(Role('subchild1'))
    roles.add(Role('subchild2'))

    order = list(roles.flatten())

    wanted = [
        'subchild0',
        'child0',
        'subchild1',
        'subchild2',
        'child1',
        'child2',
        'child3',
        'child4',
        'parent0',
        'parent1',
        'parent2',
    ]

    assert wanted == order


def test_resolve_membership():
    from ldap2pg.role import RoleSet, Role

    alice = Role('alice')
    bob = Role('bob', members=['oscar'])
    oscar = Role('oscar', parents=['alice', 'bob'])

    roles = RoleSet([alice, bob, oscar])

    roles.resolve_membership()

    assert not oscar.parents
    assert 'oscar' in alice.members

    alice.parents = ['unknown']

    with pytest.raises(ValueError):
        roles.resolve_membership()


def test_diff():
    from ldap2pg.role import Role, RoleSet

    pgmanagedroles = RoleSet([
        Role('drop-me'),
        Role('alter-me'),
        Role('nothing'),
        Role('public'),
    ])
    pgallroles = pgmanagedroles.union({
        Role('reuse-me'),
        Role('dont-touch-me'),
    })
    ldaproles = RoleSet([
        Role('reuse-me'),
        Role('alter-me', options=dict(LOGIN=True)),
        Role('nothing'),
        Role('create-me')
    ])
    queries = [
        q.args[0]
        for q in pgmanagedroles.diff(ldaproles, pgallroles)
    ]

    assert fnfilter(queries, 'ALTER ROLE "alter-me" WITH* LOGIN*;')
    assert fnfilter(queries, 'CREATE ROLE "create-me" *;')
    assert fnfilter(queries, '*DROP ROLE "drop-me";*')
    assert not fnfilter(queries, 'CREATE ROLE "reuse-me" *')
    assert not fnfilter(queries, '*nothing*')
    assert not fnfilter(queries, '*dont-touch-me*')
    assert not fnfilter(queries, '*public*')
