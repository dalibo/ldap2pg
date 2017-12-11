from __future__ import unicode_literals


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


def test_expand():
    from ldap2pg.acl import AclItem

    item = AclItem(
        acl=['ro'], dbname=AclItem.ALL_DATABASES, schema=AclItem.ALL_SCHEMAS,
    )

    assert repr(item.schema)

    items = sorted(
        item.expand(dict(
            postgres=['information_schema'],
            template1=['information_schema'],
        )),
        key=lambda x: x.dbname,
    )

    assert 2 == len(items)
    assert 'postgres' == items[0].dbname
    assert 'template1' == items[1].dbname
