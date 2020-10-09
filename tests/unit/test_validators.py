import copy

import pytest


def test_process_grant():
    from ldap2pg.validators import grantrule

    rule = grantrule(dict(
        acl='ro',
        database='postgres',
        schema='public',
        role='{cn}',
    )).as_dict()

    assert 'schemas' in rule
    assert 'databases' in rule
    assert 'privilege' in rule
    assert 'acl' not in rule
    assert 'roles' in rule

    rule = grantrule(dict(
        acl='ro',
        database='postgres',
        schema='public',
        role_attribute='cn',
    )).as_dict()

    assert 'role_attribute' not in rule
    assert '{cn}' in rule['roles']

    with pytest.raises(ValueError):
        grantrule([])

    with pytest.raises(ValueError):
        grantrule(dict(missing_privilege=True))

    with pytest.raises(ValueError):
        grantrule(dict(privilege='toto', role='toto', spurious=True))

    with pytest.raises(ValueError):
        grantrule(dict(privilege='missing role*'))


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

    rule = dict(grant=dict(privilege='rol', role='alice'))
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
        m = v[0]['grant'][0].as_dict()
        assert ['__all__'] == m['databases']
        assert ['__all__'] == m['schemas']


def test_process_syncmap_legacy():
    from ldap2pg.validators import syncmap

    grant = dict(privilege='rol', role='alice')
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
        m = v[0]['grant'][0].as_dict()
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


def test_mapping_refuse_static_rules_when_ldap():
    from ldap2pg.validators import mapping

    raw = dict(
        ldap=dict(base="toto"),
        roles=["{cn}"],
    )

    assert mapping(raw.copy())

    raw['roles'].append('static')
    with pytest.raises(ValueError):
        mapping(raw)

    raw = dict(
        ldap=dict(base="toto"),
        grant=dict(roles=["{cn}"], privilege="ro"),
    )

    assert mapping(raw.copy())

    raw['grant']['roles'].append('static')
    with pytest.raises(ValueError):
        mapping(raw)


def test_process_mapping_grant():
    from ldap2pg.validators import mapping

    mapping(dict(grant=dict(privilege='ro', role='alice')))


def test_process_mapping_ldap_join():
    from ldap2pg.validators import mapping

    v = mapping(dict(
        ldap=dict(),
        role=dict(
            name_attribute='member.sAMAccountName',
            comment='from {cn.lower()}')),
    )

    assert v['ldap']['joins']
    assert 'cn' in v['ldap']['attributes']


def test_process_mapping_ldap_compat_unexpected_dn():
    from ldap2pg.validators import mapping

    v = mapping(dict(
        ldap=dict(),
        role=dict(
            name='{cn}',
            on_unexpected_dn='ignore',
        )),
    )

    assert 'ignore' == v['ldap']['on_unexpected_dn']
    assert 'on_unexpected_dn' not in v['roles']

    # Refuse mixed on_unexpected_dn.
    with pytest.raises(ValueError):
        mapping(dict(
            ldap=dict(),
            roles=[
                dict(name='{cn}', on_unexpected_dn='ignore'),
                dict(name='{member}', on_unexpected_dn='fail'),
            ],
        ))


def test_process_ldapquery_attributes():
    from ldap2pg.validators import ldapquery, parse_scope, FormatField

    with pytest.raises(ValueError):
        ldapquery(None, None)

    with pytest.raises(ValueError):
        ldapquery(dict(base='dc=lol'), format_fields=[])

    raw = dict(
        base='dc=unit',
        scope=parse_scope('sub'),
        attribute='cn',
        on_unexpected_dn='ignore',
    )

    v = ldapquery(raw, format_fields=[])

    assert 'filter' in v
    assert ['cn'] == v['attributes']
    assert 'attribute' not in v
    assert 'ignore' == v['on_unexpected_dn']
    assert not v['allow_missing_attributes']

    with pytest.raises(ValueError):
        ldapquery(dict(raw, scope='unkqdsfq'))

    v = ldapquery(
        dict(base='o=acme'),
        [FormatField('sAMAccountName'), FormatField('dn')],
    )

    assert ['dn', 'sAMAccountName'] == sorted(v['attributes'])
    assert not v['joins']


def test_process_ldapquery_joins():
    from ldap2pg.validators import ldapquery, FormatField

    v = ldapquery(
        dict(
            base='o=acme',
            join=dict(
                member=dict(filter='(objectClass=person)'))),
        format_fields=[
            FormatField('sAMAccountName'),
            FormatField('member'),
            FormatField('member', 'cn'),
            FormatField('member', 'sAMAccountName'),
        ]
    )

    assert 'member' in v['attributes']
    assert 'sAMAccountName' in v['attributes']
    assert 'member' in v['joins']
    assert '(objectClass=person)' == v['joins']['member']['filter']
    assert ['sAMAccountName'] == v['joins']['member']['attributes']

    v = ldapquery(
        dict(base='o=acme', joins=dict(unused=dict())),
        format_fields=[
            FormatField('sAMAccountName',),
            FormatField('member', 'sAMAccountName')],
    )

    assert 'member' in v['attributes']
    assert 'sAMAccountName' in v['attributes']
    assert 'filter' in v['joins']['member']
    assert ['sAMAccountName'] == v['joins']['member']['attributes']
    assert len(v['joins']) == 1


def test_process_rolerule():
    from ldap2pg.validators import rolerule

    with pytest.raises(ValueError):
        rolerule(None)

    rule = rolerule('aline').as_dict()
    assert 'aline' == rule['names'][0]

    rule = rolerule(dict(name='rolname', parent='parent')).as_dict()
    assert ['rolname'] == rule['names']
    assert ['parent'] == rule['parents']

    with pytest.raises(ValueError):
        rolerule(dict(missing_name='noname'))

    rule = rolerule(dict(name='r', options='LOGIN SUPERUSER')).as_dict()
    assert rule['options']['LOGIN'] is True
    assert rule['options']['SUPERUSER'] is True

    rule = rolerule(dict(name='r', options=['LOGIN', 'SUPERUSER'])).as_dict()
    assert rule['options']['LOGIN'] is True
    assert rule['options']['SUPERUSER'] is True

    rule = rolerule(dict(name='r', options=['NOLOGIN', 'SUPERUSER'])).as_dict()
    assert rule['options']['LOGIN'] is False
    assert rule['options']['SUPERUSER'] is True

    with pytest.raises(ValueError) as ei:
        rolerule(dict(name='r', options='OLOLOL'))
    assert 'OLOLOL' in str(ei.value)

    rule = rolerule(dict(name_attribute='cn')).as_dict()
    assert 'name_attribute' not in rule
    assert '{cn}' in rule['names']


def test_privileges():
    from ldap2pg.validators import privileges

    with pytest.raises(ValueError):
        privileges(None)

    with pytest.raises(ValueError):
        privileges([])

    with pytest.raises(ValueError):
        privileges(dict(select=dict(iinspect_type="INSPECT")))

    with pytest.raises(ValueError):
        privileges(dict(select=None))

    raw = dict(
        __select_on_tables__=dict(
            inspect="INSPECT",
            grant="GRANT",
            revoke="REVOKE",
        ),
        ro=['__select_on_tables__'],
    )
    value = privileges(raw)
    assert raw == value


def test_verbosity():
    from ldap2pg.validators import verbosity

    assert 'WARNING' == verbosity('WARNING')
    assert 'DEBUG' == verbosity([10])
    assert 'CRITICAL' == verbosity([4, 1, -10])

    with pytest.raises(ValueError):
        verbosity('TOTO')


def test_shared_queries():
    from ldap2pg.validators import shared_queries

    with pytest.raises(ValueError):
        shared_queries(['toto'])

    with pytest.raises(ValueError):
        shared_queries({'toto': {'not': 'string'}})

    assert {} == shared_queries(None)
    assert {} == shared_queries([])
    assert {'toto': 'SELECT 1;'} == shared_queries({'toto': 'SELECT 1;'})
