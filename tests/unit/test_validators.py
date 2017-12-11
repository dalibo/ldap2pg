import pytest


def test_process_acldict():
    from ldap2pg.validators import acldict

    with pytest.raises(ValueError):
        acldict([])

    acl_dict = acldict(dict(ro=dict(inspect='SQL', grant='SQL', revoke='SQL')))

    assert 'ro' in acl_dict


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

    grantrule(dict(
        acl='ro',
        database='postgres',
        schema='public',
        role_attribute='cn',
    ))


def test_ismapping():
    from ldap2pg.validators import ismapping

    assert ismapping(dict(ldap=dict()))
    assert ismapping(dict(roles=[]))
    assert ismapping(dict(role=dict()))
    assert not ismapping([])
    assert not ismapping(dict(__all__=[]))


def test_process_syncmap():
    from ldap2pg.validators import syncmap

    fixtures = [
        # Canonical case.
        dict(
            __all__=dict(
                __any__=[
                    dict(role=dict(name='alice')),
                ]
            ),
        ),
        # Squeeze list.
        dict(
            __all__=dict(
                __any__=dict(role=dict(name='alice')),
            ),
        ),
        # Squeeze also schema.
        dict(__all__=dict(role=dict(name='alice'))),
        # Squeeze also database.
        dict(role=dict(name='alice')),
        # Direct list (this is 1.0 format).
        [dict(role=dict(name='alice'))],
    ]

    for raw in fixtures:
        v = syncmap(raw)

        assert isinstance(v, dict)
        assert '__all__' in v
        assert '__any__' in v['__all__']
        maplist = v['__all__']['__any__']
        assert 1 == len(maplist)
        assert 'roles' in maplist[0]

    # Missing rules
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
    from ldap2pg.validators import ldapquery, parse_scope

    raw = dict(base='dc=unit', scope=parse_scope('sub'), attribute='cn')

    v = ldapquery(raw)

    assert 'attributes' in v
    assert 'attribute' not in v
    assert 'filter' in v

    with pytest.raises(ValueError):
        ldapquery(dict(raw, scope='unkqdsfq'))


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
