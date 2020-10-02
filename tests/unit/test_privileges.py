# -*- coding: utf-8 -*-

from __future__ import unicode_literals

from fnmatch import filter as fnfilter

import pytest


def test_privilege_object():
    from ldap2pg.privilege import Privilege

    connect = Privilege('connect')
    ro = Privilege('ro')

    assert 'connect' in repr(connect)
    assert connect < ro


def test_grant_object():
    from ldap2pg.privilege import Privilege, Grant
    from ldap2pg.role import Role

    priv = Privilege(name='connect', grant='GRANT {database} TO {role};')
    item = Grant(priv.name, dbname='backend', schema=None, role='daniel')
    qry = priv.grant(item)

    assert 'GRANT "backend"' in qry.args[0]
    assert 'daniel' in qry.args[0]

    assert 'db' in repr(Grant('p', ['db'], ['schema']))

    # Test hash with Role object.
    str_h = hash(Grant('priv', ['db'], ['schema'], role=Role(u'rôle')))
    obj_h = hash(Grant('priv', ['db'], ['schema'], role=u'rôle'))
    assert str_h == obj_h


def test_grant_set():
    from ldap2pg.privilege import Grant, Acl

    acl0 = Grant('ro', 'postgres', None, 'alice')
    acl1 = Grant.from_row('ro', 'postgres', None, 'bob')
    assert acl0 < acl1

    duplicata = Grant('ro', 'postgres', None, 'alice')
    set_ = Acl([acl0, acl1, duplicata])

    assert 2 == len(set_)
    assert 'postgres' in str(acl0)
    assert 'ro' in repr(acl0)
    assert acl0 != acl1
    assert acl0 == duplicata


def test_revoke():
    from ldap2pg.privilege import Privilege, Grant

    priv = Privilege(name='connect', revoke='REVOKE {database} FROM {role};')
    item = Grant(priv.name, dbname='backend', schema=None, role='daniel')
    qry = priv.revoke(item)

    assert 'REVOKE "backend"' in qry.args[0]
    assert 'daniel' in qry.args[0]


def test_expand_defacl():
    from ldap2pg.privilege import DefAcl, Acl, Grant, UserError

    priv = DefAcl('select', grant='ALTER FOR GRANT SELECT')
    item0 = Grant('select', Grant.ALL_DATABASES, schema=Grant.ALL_SCHEMAS)
    item1 = Grant('select', ['postgres'], schema=['information_schema'])

    assert repr(item0.schema)

    set_ = Acl([item0, item1])

    items = sorted(
        set_.expandgrants(
            aliases=dict(select=['select']),
            privileges={priv.name: priv},
            databases=dict(
                postgres=dict(
                    information_schema=['postgres'],
                ),
                template1=dict(
                    information_schema=['postgres'],
                ),
            ),
        ),
        key=lambda x: x.dbname,
    )

    assert 3 == len(items)
    assert 'postgres' == items[0].dbname
    assert 'template1' == items[2].dbname

    with pytest.raises(UserError):
        list(set_.expandgrants(
            aliases=dict(select=['select']),
            privileges={priv.name: priv},
            databases=dict(),
        ))


def test_expand_datacl():
    from ldap2pg.privilege import DatAcl, Grant

    priv = DatAcl('c', grant='GRANT CONNECT')
    item = Grant('c', dbname=Grant.ALL_DATABASES, schema=None)

    items = sorted(priv.expand(
        item, databases=dict(postgres=0xbad, template1="ignored value"),
    ),    key=lambda x: x.dbname)

    assert 2 == len(items)
    assert 'postgres' == items[0].dbname
    assert items[0].schema is None
    assert 'template1' == items[1].dbname
    assert items[1].schema is None


def test_expand_global_defacl():
    from ldap2pg.privilege import GlobalDefAcl, Grant

    priv = GlobalDefAcl('c', grant='GRANT CONNECT')
    item = Grant('c', dbname=Grant.ALL_DATABASES, schema=None)

    items = sorted(priv.expand(
        item, databases=dict(postgres=dict(public=['postgres', 'admin'])),
    ), key=lambda x: x.owner)

    assert 2 == len(items)
    item = items[0]
    assert 'postgres' == item.dbname
    assert item.schema is None
    assert 'admin' == item.owner

    item = items[1]
    assert 'postgres' == item.dbname
    assert item.schema is None
    assert 'postgres' == item.owner


def test_expand_nok():
    from ldap2pg.privilege import Acl, Grant

    set_ = Acl([Grant('inexistant')])

    with pytest.raises(ValueError):
        list(set_.expandgrants(
            aliases=dict(),
            privileges=dict(),
            databases=dict(),
        ))

    set_ = Acl([Grant('inexistant_dep')])

    with pytest.raises(ValueError):
        list(set_.expandgrants(
            aliases=dict(inexistant_dep=['inexistant']),
            privileges=dict(),
            databases=dict(),
        ))


def test_check_groups():
    from ldap2pg.privilege import check_group_definitions

    privileges = dict(group=['inexistant'])

    with pytest.raises(ValueError):
        check_group_definitions(dict(), privileges)


def test_diff(mocker):
    from ldap2pg.privilege import Privilege, Grant, Acl

    priv = Privilege(name='priv', revoke='REVOKE {role}', grant='GRANT {role}')
    nogrant = Privilege(name='nogrant', revoke='REVOKE')
    norvk = Privilege(name='norvk', grant='GRANT')
    privileges = {p.name: p for p in [priv, nogrant, norvk]}

    item0 = Grant(privilege=priv.name, dbname='backend', role='daniel')
    pgacl = Acl([
        item0,
        Grant(privilege=priv.name, dbname='backend', role='alice'),
        Grant(priv.name, dbname='backend', role='irrelevant', full=None),
        Grant(privilege=norvk.name, role='torevoke'),
    ])
    ldapacl = Acl([
        item0,
        Grant(privilege=priv.name, dbname='backend', role='david'),
        Grant(privilege=nogrant.name, role='togrant'),
    ])

    queries = [q.args[0] for q in pgacl.diff(ldapacl, privileges)]

    assert not fnfilter(queries, 'REVOKE "daniel"*')
    assert fnfilter(queries, 'REVOKE "alice"*')
    assert fnfilter(queries, 'GRANT "david"*')


def test_make_privilege_shared():
    from ldap2pg.defaults import make_privilege

    kwargs = dict(
        tpl=dict(
            inspect=dict(shared_query='datacl', keys=['%(privilege)s']),
            grant='GRANT %(TYPE)s;', revoke='REVOK %(TYPE)s;',
        ),
        name='__connect__',
        TYPE='DATABASE',
        privilege='connect',
    )

    name, priv = make_privilege(**kwargs)
    assert ['CONNECT'] == priv['inspect']['keys']

    with pytest.raises(Exception):
        make_privilege(**dict(
            kwargs,
            tpl=dict(
                kwargs['tpl'],
                inspect=dict(shared_query='badacl', keys=['toto']),
            ),
        ))


def test_grant_rule():
    from ldap2pg.privilege import GrantRule

    r = GrantRule(
        privilege='{extensionAttribute0}',
        databases=['{extensionAttribute1}'],
        schemas=['{cn}'],
        roles=['{cn}'],
    )

    map_ = r.attributes_map
    assert '__self__' in map_
    assert 'extensionAttribute0' in map_['__self__']
    assert 'extensionAttribute1' in map_['__self__']
    assert 'cn' in map_['__self__']

    assert repr(r)

    d = r.as_dict()
    assert '{extensionAttribute0}' == d['privilege']
    assert ['{extensionAttribute1}'] == d['databases']
    assert ['{cn}'] == d['schemas']
    assert ['{cn}'] == d['roles']

    vars_ = dict(__self__=[dict(
        cn=['rol0', 'rol1'],
        extensionAttribute0=['ro'],
        extensionAttribute1=['appdb'],
    )])
    grants = list(r.generate(vars_))
    assert 2 == len(grants)


def test_grant_rule_match():
    from ldap2pg.privilege import GrantRule

    r = GrantRule(
        privilege='ro',
        databases=['mydb'],
        schemas=None,
        roles=['{cn}'],
        role_match='prefix_*',
    )

    vars_ = dict(__self__=[dict(
        cn=['prefix_rol0', 'ignored', 'prefix_rol1'],
    )])
    grants = list(r.generate(vars_))
    assert 2 == len(grants)


def test_grant_rule_all_databases():
    from ldap2pg.privilege import GrantRule, Grant

    r = GrantRule(
        privilege='ro',
        databases=['__all__'],
        schemas=None,
        roles=['role'],
    )

    vars_ = dict(__self__=[dict(dn=['dn'])])
    grant, = r.generate(vars_)
    assert grant.dbname is Grant.ALL_DATABASES
