from __future__ import unicode_literals

from fnmatch import filter as fnfilter

import pytest


def test_role():
    from ldap2pg.role import Role, RoleOptions

    role = Role(name='toto')

    assert 'toto' == role.name
    assert 'toto' == str(role)
    assert 'toto' in repr(role)

    roles = sorted([Role('b'), Role('a')])

    assert ['a', 'b'] == roles

    row = ['name', ('member0',)]
    row += [True] * len(RoleOptions.SUPPORTED_COLUMNS)
    row += ['Managed by ldap2pg.']
    role = Role.from_row(*row)

    assert 'Managed by ldap2pg.' == role.comment


def test_create():
    from ldap2pg.role import Role

    role = Role(name='toto', members=['titi'], comment='Mycom')

    queries = [q.args[0] for q in role.create()]

    assert fnfilter(queries, 'CREATE ROLE "toto" *;')
    assert fnfilter(queries, 'GRANT "toto" TO "titi";')
    assert fnfilter(queries, '*Mycom*')


def test_alter():
    from ldap2pg.role import Role

    a = Role(name='toto', members=['titi'], options=dict(LOGIN=True))
    b = Role(name='toto', members=['tata'], options=dict(LOGIN=False))

    queries = [q.args[0] for q in a.alter(a)]
    assert not queries

    queries = [q.args[0] for q in a.alter(b)]

    assert fnfilter(queries, 'ALTER ROLE "toto" *;')
    assert fnfilter(queries, 'GRANT "toto" TO "tata";')
    assert fnfilter(queries, 'REVOKE "toto" FROM "titi";')


def test_drop():
    from ldap2pg.inspector import Database
    from ldap2pg.role import Role

    role = Role(name='toto', members=['titi'])

    db = Database('postgres', 'postgres')
    queries = [q.args[0] for q in role.drop(databases=[db])]

    assert fnfilter(queries, '*pg_terminate_backend*')
    assert fnfilter(queries, '*REASSIGN OWNED*TO "postgres";*')
    assert fnfilter(queries, '*DROP OWNED BY "toto";')
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


def test_comment():
    from ldap2pg.role import Role

    a = Role(name='alan')
    b = Role(name='alan', comment='New comment')
    queries = [q.args[0] for q in a.alter(b)]

    assert 'COMMENT ON ROLE "alan"' in queries[0]


def test_rename_query():
    from ldap2pg.role import Role

    a = Role(name='alan')
    b = Role(name='Alan')
    queries = [q.args[0] for q in a.rename(b)]

    assert '"alan" RENAME TO "Alan"' in queries[0]


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
        Role('rename-me'),
    ])
    pgallroles = pgmanagedroles.union({
        Role('reuse-me'),
        Role('dont-touch-me'),
    })
    ldaproles = RoleSet([
        Role('reuse-me'),
        Role('alter-me', options=dict(LOGIN=True)),
        Role('nothing'),
        Role('create-me'),
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


def test_diff_rename():
    from ldap2pg.role import Role, RoleSet

    pgmanagedroles = RoleSet([
        Role('min2min'),
        Role('min2mix'),
        Role('min2maj'),
    ])
    pgallroles = pgmanagedroles.union({
        Role('Mix2Min'),
        Role('Mix2Mix'),
        Role('Mix2Maj'),
        Role('MAJ2MIN'),
        Role('MAJ2MIX'),
        Role('MAJ2MAJ'),
    })
    ldaproles = RoleSet([
        Role('min2min'),
        Role('Min2Mix'),
        Role('MIN2MAJ'),
        Role('mix2min'),
        Role('Mix2Mix'),
        Role('MIX2MAJ'),
        Role('maj2min'),
        Role('Maj2Mix'),
        Role('MAJ2MAJ'),
    ])
    queries = [
        q.args[0]
        for q in pgmanagedroles.diff(ldaproles, pgallroles)
    ]

    assert fnfilter(queries, '*"MAJ2MIX" RENAME TO "Maj2Mix";')
    assert fnfilter(queries, '*"MAJ2MIN" RENAME TO "maj2min";')
    assert fnfilter(queries, '*"min2mix" RENAME TO "Min2Mix";')
    assert fnfilter(queries, '*"min2maj" RENAME TO "MIN2MAJ";')
    assert fnfilter(queries, '*"Mix2Maj" RENAME TO "MIX2MAJ";')
    assert fnfilter(queries, '*"Mix2Min" RENAME TO "mix2min";')
    assert not fnfilter(queries, '*CREATE ROLE*')


def test_diff_rename_members():
    from ldap2pg.role import Role, RoleSet

    pgmanagedroles = RoleSet()
    pgallroles = pgmanagedroles.union({
        Role('parent', members=['min2mix', 'min2min']),
        Role('min2mix'),
        Role('min2min'),

    })
    ldaproles = RoleSet([
        Role('parent', members=['Min2Mix', 'min2min']),
        Role('Min2Mix'),
        Role('min2min'),
    ])
    queries = [
        q.args[0]
        for q in pgmanagedroles.diff(ldaproles, pgallroles)
    ]

    # Don't modify membership.
    assert not fnfilter(queries, '*GRANT*')
    assert not fnfilter(queries, '*REVOKE*')


def test_diff_not_rename():
    from ldap2pg.role import Role, RoleSet

    pgmanagedroles = RoleSet()
    pgallroles = pgmanagedroles.union({
        Role('ambigue_from'),
        Role('AMBIGUE_FROM'),
        Role('Ambigue_To'),
        Role('bothmin'),
        Role('BothMix'),
    })
    ldaproles = RoleSet([
        Role('Ambigue_From'),
        Role('ambigue_to'),
        Role('AMBIGUE_TO'),
        Role('bothmin'),
        Role('Bothmin'),
        Role('BothMix'),
        Role('bothmix'),
    ])
    queries = [
        q.args[0]
        for q in pgmanagedroles.diff(ldaproles, pgallroles)
    ]

    assert not fnfilter(queries, '* RENAME TO *')
    assert fnfilter(queries, '*CREATE ROLE "Ambigue_From"*')
    assert fnfilter(queries, '*CREATE ROLE "ambigue_to"*')
    assert fnfilter(queries, '*CREATE ROLE "AMBIGUE_TO"*')


def test_rule():
    from ldap2pg.role import RoleRule

    r = RoleRule(
        names=['static', 'prefix_{cn}', '{uid}_{member}'],
        parents=['{uid}', '{member.cn}'],
        members=[],
        comment='From {dn}',
        options={'SUPERUSER': True},
    )

    map_ = r.attributes_map
    assert '__self__' in map_
    assert 'uid' in map_['__self__']
    assert 'member' not in map_['__self__']
    assert 'member' in map_

    assert 5 == len(r.all_fields)
    assert repr(r)

    d = r.as_dict()
    assert 'SUPERUSER' in d['options']
    assert ['static', 'prefix_{cn}', '{uid}_{member}'] == d['names']
    assert [] == d['members']
    assert ['{uid}', '{member.cn}'] == d['parents']
    assert 'From {dn}' == d['comment']

    vars_ = dict(
        __self__=[dict(
            dn=['cn=group,ou=groups'],
            cn=['cn'],
            uid=['uid'],
        )],
        member=[
            dict(
                dn=['cn=m0'],
                cn=['m0'],
            ),
            dict(
                dn=['cn=m1'],
                cn=['m1'],
            ),
        ],
    )

    roles = list(r.generate(vars_))
    assert 4 == len(roles)


def test_role_rule_dynamic_comments():
    from ldap2pg.role import RoleRule

    r = RoleRule(
        names=['{member}'],
        comment='From {member}',
    )

    vars_ = dict(__self__=[dict(
        dn=['cn=group,ou=groups'],
        member=['m0', 'm1'],
    )])

    roles = list(r.generate(vars_))

    assert 2 == len(roles)
    for role in roles:
        assert role.name in role.comment


def test_role_rule_too_many_comments():
    from ldap2pg.role import RoleRule, CommentError

    r = RoleRule(
        names=['{member}'],
        comment='From {more}',
    )

    vars_ = dict(__self__=[dict(
        dn=['cn=group,ou=groups'],
        member=['m0', 'm1'],
        more=['0', '1', '2'],
    )])

    with pytest.raises(CommentError):
        list(r.generate(vars_))


def test_role_rule_no_comment():
    from ldap2pg.role import RoleRule, CommentError

    r = RoleRule(
        names=['{member}'],
        comment='From {desc}',
    )

    vars_ = dict(__self__=[dict(
        dn=['cn=group,ou=groups'],
        desc=[],
        member=['m0', 'm1'],
    )])

    with pytest.raises(CommentError):
        list(r.generate(vars_))


def test_role_rule_not_enough_comment():
    from ldap2pg.role import RoleRule, CommentError

    r = RoleRule(
        names=['{member}'],
        comment='From {less}',
    )

    vars_ = dict(__self__=[dict(
        dn=['cn=group,ou=groups'],
        member=['m0', 'm1', 'm2'],
        less=['l0', 'l1']
    )])

    with pytest.raises(CommentError):
        list(r.generate(vars_))
