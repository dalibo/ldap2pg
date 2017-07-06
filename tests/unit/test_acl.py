from __future__ import unicode_literals

from fnmatch import filter as fnfilter


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


def test_revoke():
    from ldap2pg.acl import Acl, AclItem

    acl = Acl(name='connect', revoke='REVOKE %(database)s FROM %(role)s;')
    item = AclItem(acl=acl.name, dbname='backend', schema=None, role='daniel')
    queries = [q.args[0] for q in acl.revoke(item)]

    assert 1 == len(queries)
    assert fnfilter(queries, '*backend*')
    assert fnfilter(queries, '*daniel*')
