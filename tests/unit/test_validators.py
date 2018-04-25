import copy

import pytest


def test_process_grant():
    from ldap2pg.validators import grantrule

    with pytest.raises(ValueError):
        grantrule([])

    with pytest.raises(ValueError):
        grantrule(dict(missing_acl=True))

    with pytest.raises(ValueError):
        grantrule(dict(acl='toto', spurious_attribute=True))

    with pytest.raises(ValueError):
        grantrule(dict(acl='missing role*'))

    rule = grantrule(dict(
        acl='ro',
        database='postgres',
        schema='public',
        role_attribute='cn',
    ))

    assert 'schemas' in rule
    assert 'databases' in rule


def test_ismapping():
    from ldap2pg.validators import ismapping

    assert ismapping(dict(ldap=dict()))
    assert ismapping(dict(roles=[]))
    assert ismapping(dict(role=dict()))
    assert ismapping(dict(grant=dict()))
    assert not ismapping([])
    assert not ismapping(dict(__all__=[]))


def test_process_syncmap():
    from ldap2pg.validators import syncmap

    assert [] == syncmap(None)

    rule = dict(grant=dict(acl='rol', role='alice'))
    fixtures = [
        # Canonical case.
        [rule],
        # full map dict (ldap2pg 2 format).
        dict(__all__=dict(__all__=[rule])),
        # Squeeze inner list.
        dict(__all__=dict(__all__=rule)),
        # Squeeze also schema.
        dict(__all__=rule),
        # Squeeze also database.
        rule,
    ]

    for raw in fixtures:
        v = syncmap(copy.deepcopy(raw))

        assert isinstance(v, list)
        assert 1 == len(v)
        assert 'grant' in v[0]
        m = v[0]['grant'][0]
        assert '__all__' == m['databases']
        assert '__all__' == m['schemas']


def test_process_syncmap_legacy():
    from ldap2pg.validators import syncmap

    grant = dict(acl='rol', role='alice')
    fixtures = [
        dict(db=dict(schema=dict(grant=grant))),
        dict(db=dict(grant=dict(schema='schema', **grant))),
        dict(grant=dict(database='db', schema='schema', **grant)),
    ]

    for raw in fixtures:
        v = syncmap(copy.deepcopy(raw))

        assert isinstance(v, list)
        assert 1 == len(v)
        assert 'grant' in v[0]
        m = v[0]['grant'][0]
        assert ['db'] == m['databases']
        assert ['schema'] == m['schemas']


def test_process_syncmap_bad():
    from ldap2pg.validators import syncmap

    raw = dict(ldap=dict(base='dc=unit', attribute='cn'))
    with pytest.raises(ValueError):
        syncmap(raw)

    bad_fixtures = [
        'string_value',
        [None],
    ]
    for raw in bad_fixtures:
        with pytest.raises(ValueError):
            syncmap(raw)


def test_process_mapping_grant():
    from ldap2pg.validators import mapping

    mapping(dict(grant=dict(acl='ro', role='alice')))


def test_process_ldapquery():
    from ldap2pg.validators import mapping, ldapquery, parse_scope

    with pytest.raises(ValueError):
        ldapquery(None)

    raw = dict(base='dc=unit', scope=parse_scope('sub'), attribute='cn')

    v = ldapquery(raw)

    assert 'filter' in v

    with pytest.raises(ValueError):
        ldapquery(dict(raw, scope='unkqdsfq'))

    v = mapping(dict(role=dict(name_attribute='cn'), ldap=dict(base='o=acme')))

    assert ['cn'] == v['ldap']['attributes']


def test_process_rolerule():
    from ldap2pg.validators import rolerule

    with pytest.raises(ValueError):
        rolerule(None)

    rule = rolerule('aline')
    assert 'aline' == rule['names'][0]

    rule = rolerule(dict(name='rolname', parent='parent'))
    assert ['rolname'] == rule['names']
    assert ['parent'] == rule['parents']

    with pytest.raises(ValueError):
        rolerule(dict(missing_name='noname'))

    rule = rolerule(dict(name='r', options='LOGIN SUPERUSER'))
    assert rule['options']['LOGIN'] is True
    assert rule['options']['SUPERUSER'] is True

    rule = rolerule(dict(name='r', options=['LOGIN', 'SUPERUSER']))
    assert rule['options']['LOGIN'] is True
    assert rule['options']['SUPERUSER'] is True

    rule = rolerule(dict(name='r', options=['NOLOGIN', 'SUPERUSER']))
    assert rule['options']['LOGIN'] is False
    assert rule['options']['SUPERUSER'] is True

    with pytest.raises(ValueError) as ei:
        rolerule(dict(name='r', options='OLOLOL'))
    assert 'OLOLOL' in str(ei.value)


def test_acls():
    from ldap2pg.validators import acls

    with pytest.raises(ValueError):
        acls(None)

    with pytest.raises(ValueError):
        acls([])

    with pytest.raises(ValueError):
        acls(dict(select=dict(iinspect_type="INSPECT")))

    with pytest.raises(ValueError):
        acls(dict(select=None))

    raw = dict(
        __select_on_tables__=dict(
            inspect="INSPECT",
            grant="GRANT",
            revoke="REVOKE",
        ),
        ro=['__select_on_tables__'],
    )
    value = acls(raw)
    assert raw == value
