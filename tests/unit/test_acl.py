from __future__ import unicode_literals

import pytest


def test_acl():
    from ldap2pg.acl import Acl

    connect = Acl('connect')
    ro = Acl('ro')

    assert 'connect' in repr(connect)
    assert connect < ro


def test_items():
    from ldap2pg.acl import AclItem, AclSet

    acl0 = AclItem('ro', 'postgres', None, 'alice')
    acl1 = AclItem.from_row('ro', 'postgres', None, 'bob')
    assert acl0 < acl1

    duplicata = AclItem('ro', 'postgres', None, 'alice')
    set_ = AclSet([acl0, acl1, duplicata])

    assert 2 == len(set_)
    assert 'postgres' in str(acl0)
    assert 'ro' in repr(acl0)
    assert acl0 != acl1
    assert acl0 == duplicata


def test_grant():
    from ldap2pg.acl import Acl, AclItem

    acl = Acl(name='connect', grant='GRANT {database} TO {role};')
    item = AclItem(acl=acl.name, dbname='backend', schema=None, role='daniel')
    qry = acl.grant(item)

    assert 'GRANT "backend"' in qry.args[0]
    assert 'daniel' in qry.args[0]


def test_revoke():
    from ldap2pg.acl import Acl, AclItem

    acl = Acl(name='connect', revoke='REVOKE {database} FROM {role};')
    item = AclItem(acl=acl.name, dbname='backend', schema=None, role='daniel')
    qry = acl.revoke(item)

    assert 'REVOKE "backend"' in qry.args[0]
    assert 'daniel' in qry.args[0]


def test_expand_defacl():
    from ldap2pg.acl import DefAcl, AclSet, AclItem

    acl = DefAcl('select', grant='ALTER FOR GRANT SELECT')
    item0 = AclItem(
        acl='select', dbname=AclItem.ALL_DATABASES, schema=AclItem.ALL_SCHEMAS,
    )
    item1 = AclItem(
        acl='select', dbname='postgres', schema='public',
    )

    assert repr(item0.schema)

    set_ = AclSet([item0, item1])

    items = sorted(
        set_.expanditems(
            aliases=dict(select=['select']),
            acl_dict={acl.name: acl},
            databases=dict(
                postgres=['information_schema'],
                template1=['information_schema'],
            ),
            owners=['postgres'],
        ),
        key=lambda x: x.dbname,
    )

    assert 3 == len(items)
    assert 'postgres' == items[0].dbname
    assert 'template1' == items[2].dbname


def test_expand_datacl():
    from ldap2pg.acl import DatAcl, AclItem

    acl = DatAcl('c', grant='GRANT CONNECT')
    item = AclItem(acl='c', dbname=AclItem.ALL_DATABASES, schema=None)

    items = sorted(acl.expand(
        item, databases=dict(postgres=0xbad, template1="ignored value"),
    ),    key=lambda x: x.dbname)

    assert 2 == len(items)
    assert 'postgres' == items[0].dbname
    assert items[0].schema is None
    assert 'template1' == items[1].dbname
    assert items[1].schema is None


def test_expand_global_defacl():
    from ldap2pg.acl import GlobalDefAcl, AclItem

    acl = GlobalDefAcl('c', grant='GRANT CONNECT')
    item = AclItem(acl='c', dbname=AclItem.ALL_DATABASES, schema=None)

    items = sorted(acl.expand(
        item, databases=dict(postgres=0xbad),
        owners=['postgres', 'admin'],
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
    from ldap2pg.acl import AclSet, AclItem

    set_ = AclSet([AclItem('inexistant')])

    with pytest.raises(ValueError):
        list(set_.expanditems(
            aliases=dict(),
            acl_dict=dict(),
            databases=dict(),
            owners=[],
        ))

    set_ = AclSet([AclItem('inexistant_dep')])

    with pytest.raises(ValueError):
        list(set_.expanditems(
            aliases=dict(inexistant_dep=['inexistant']),
            acl_dict=dict(),
            databases=dict(),
            owners=[],
        ))


def test_check_groups():
    from ldap2pg.acl import check_group_definitions

    acls = dict(group=['inexistant'])

    with pytest.raises(ValueError):
        check_group_definitions(dict(), acls)
