def test_set():
    from ldap2pg.acl import AclItem, AclSet

    acl0 = AclItem('ro', 'postgres', None, 'alice')
    acl1 = AclItem.from_row('ro', 'postgres', None, 'bob')
    duplicata = AclItem('ro', 'postgres', None, 'alice')
    set_ = AclSet([acl0, acl1, duplicata])

    assert 2 == len(set_)
    assert 'postgres' in str(acl0)
    assert 'ro' in repr(acl0)
    assert acl0 != acl1
    assert acl0 == duplicata
