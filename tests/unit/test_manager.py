from __future__ import unicode_literals

import pytest


def test_query_ldap(mocker):
    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.return_value = [
        ('dn=a', {}),
        ('dn=b', {}),
        (None, {'ref': True}),
    ]

    entries = manager.query_ldap(
        base='ou=people,dc=global', filter='(objectClass=*)',
        scope=2, attributes=['cn'],
    )

    assert 2 == len(entries)

    manager.ldapconn.search_s.return_value = [('dn=a', {'a': b'\xbb'})]
    with pytest.raises(UserError):
        manager.query_ldap(
            base='ou=people,dc=global', filter='(objectClass=*)',
            scope=2, attributes=['cn'],
        )


def test_query_ldap_bad_filter(mocker):
    from ldap2pg.manager import SyncManager, LDAPError, UserError

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.side_effect = LDAPError()

    with pytest.raises(UserError):
        manager.query_ldap(
            base='dc=unit', filter='(broken', scope=2, attributes=[],
        )

    assert manager.ldapconn.search_s.called is True


def test_process_entry_static():
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    roles = manager.process_ldap_entry(
        entry=('dn',), names=['ALICE'], parents=['postgres'],
        options=dict(LOGIN=True), comment='Custom.',
    )
    roles = list(roles)

    assert 1 == len(roles)
    assert 'alice' in roles
    assert 'postgres' in roles[0].parents
    assert 'Custom.' == roles[0].comment


def test_process_entry_user():
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = ('dn', {'cn': ['alice', 'bob']})

    roles = manager.process_ldap_entry(
        entry, names=['{cn}'],
        options=dict(LOGIN=True),
    )
    roles = list(roles)

    assert 2 == len(roles)
    assert 'alice' in roles
    assert 'bob' in roles
    assert roles[0].options['LOGIN'] is True


def test_process_entry_dn():
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = ('dn', {'member': ['cn=alice,dc=unit', 'cn=bob,dc=unit']})

    roles = manager.process_ldap_entry(entry, names=['{member.cn}'])
    roles = list(roles)
    names = {r.name for r in roles}

    assert 2 == len(roles)
    assert 'alice' in names
    assert 'bob' in names


def test_process_entry_membership(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entries = [
        ('cn=group0', {
            'cn': ['group0'],
            'member': ['cn=alice,dc=unit', 'cn=alain,dc=unit']}),
        ('cn=group1', {
            'cn': ['group1'],
            'member': ['cn=bob,dc=unit', 'cn=benoit,dc=unit']}),
    ]

    roles = []
    rule = dict(
        members=['{member.cn}'],
        parents=['{cn}'],
    )
    for i, entry in enumerate(entries):
        name = 'role%d' % i
        roles += list(manager.process_ldap_entry(
            entry, names=[name], **rule))

    assert 2 == len(roles)
    assert 'alice' in roles[0].members
    assert 'alain' in roles[0].members
    assert 'bob' not in roles[0].members
    assert 'benoit' not in roles[0].members
    assert 'group0' in roles[0].parents
    assert 'group1' not in roles[0].parents

    assert 'alice' not in roles[1].members
    assert 'alain' not in roles[1].members
    assert 'bob' in roles[1].members
    assert 'benoit' in roles[1].members
    assert 'group0' not in roles[1].parents
    assert 'group1' in roles[1].parents


def test_apply_role_rule_ko(mocker):
    gla = mocker.patch('ldap2pg.manager.expand_attributes', autospec=True)

    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager()

    gla.side_effect = ValueError
    items = manager.apply_role_rules(
        entries=[('dn0',), ('dn1',)],
        rules=[dict(names=['{cn}'])],
    )
    with pytest.raises(UserError):
        list(items)


def test_apply_role_rule_unexpected_dn(mocker):
    gla = mocker.patch('ldap2pg.manager.expand_attributes', autospec=True)

    from ldap2pg.manager import SyncManager, RDNError, UserError

    manager = SyncManager()

    gla.side_effect = RDNError

    list(manager.apply_role_rules(
        entries=[('dn0',), ('dn1',)],
        rules=[dict(names=['{cn}'], on_unexpected_dn='warn')],
    ))

    list(manager.apply_role_rules(
        entries=[('dn0',), ('dn1',)],
        rules=[dict(names=['{cn}'], on_unexpected_dn='ignore')],
    ))

    with pytest.raises(UserError):
        list(manager.apply_role_rules(
            entries=[('dn0',), ('dn1',)],
            rules=[dict(names=['{cn}'])],
        ))


def test_apply_grant_rule_ok(mocker):
    gla = mocker.patch('ldap2pg.manager.expand_attributes', autospec=True)

    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    gla.side_effect = [['alice'], ['bob']]
    items = manager.apply_grant_rules(
        grant=[dict(
            privilege='connect',
            databases=['postgres'],
            schemas='__any__',
            roles=['{cn}'],
        )],
        entries=[None, None],
    )
    items = list(items)
    assert 2 == len(items)
    assert 'alice' == items[0].role
    assert 'postgres' == items[0].dbname[0]
    # Ensure __any__ schema is mapped to None
    assert items[0].schema is None
    assert 'bob' == items[1].role


def test_apply_grant_rule_wrong_attr(mocker):
    gla = mocker.patch('ldap2pg.manager.expand_attributes')

    from ldap2pg.manager import SyncManager, UserError

    gla.side_effect = ValueError('POUET')
    items = SyncManager().apply_grant_rules(
        grant=[dict(roles=['{cn}'])],
        entries=[None, None],
    )
    with pytest.raises(UserError):
        list(items)


def test_apply_grant_rule_all_schema(mocker):
    gla = mocker.patch('ldap2pg.manager.expand_attributes', autospec=True)

    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    gla.side_effect = [['alice']]
    items = manager.apply_grant_rules(
        grant=[dict(
            privilege='connect',
            databases=['postgres'],
            schema='__all__',
            roles=['{cn}'],
        )],
        entries=[None],
    )
    items = list(items)
    assert 1 == len(items)
    assert 'alice' == items[0].role
    assert 'postgres' == items[0].dbname[0]
    # Ensure __all__ schema is mapped to object
    assert items[0].schema != '__all__'


def test_apply_grant_rule_filter(mocker):
    from ldap2pg.manager import SyncManager

    items = SyncManager().apply_grant_rules(
        grant=[dict(
            privilege='connect',
            database='postgres',
            schema='__any__',
            role_match='*_r',
            roles=['alice_r', 'bob_rw'],
        )],
        entries=[None],
    )
    items = list(items)
    assert 1 == len(items)
    assert 'alice_r' == items[0].role


def test_apply_grant_rule_nodb(mocker):
    gla = mocker.patch('ldap2pg.manager.expand_attributes', autospec=True)

    from ldap2pg.manager import Grant, SyncManager

    manager = SyncManager()

    gla.return_value = ['alice']
    items = list(manager.apply_grant_rules(
        grant=[dict(
            privilege='connect',
            database='__all__', schema='__any__',
            roles=['{cn}'],
        )],
        entries=[None],
    ))
    assert items[0].dbname is Grant.ALL_DATABASES


def test_inspect_ldap_grants(mocker):
    la = mocker.patch(
        'ldap2pg.manager.SyncManager.apply_grant_rules', autospec=True)

    from ldap2pg.manager import SyncManager, Grant
    from ldap2pg.privilege import NspAcl
    from ldap2pg.utils import make_group_map

    privileges = dict(ro=NspAcl(name='ro'))
    la.return_value = [Grant('ro', 'postgres', None, 'alice')]

    manager = SyncManager(
        psql=mocker.Mock(), ldapconn=mocker.Mock(), privileges=privileges,
        privilege_aliases=make_group_map(privileges)
    )
    syncmap = [dict(roles=[], grant=dict(privilege='ro'))]

    _, grants = manager.inspect_ldap(syncmap=syncmap)

    assert 1 == len(grants)


def test_postprocess_grants():
    from ldap2pg.manager import SyncManager, Grant, Acl
    from ldap2pg.privilege import DefAcl

    manager = SyncManager(
        privileges=dict(ro=DefAcl(name='ro')),
        privilege_aliases=dict(ro=['ro']),
    )

    # No owners
    acl = manager.postprocess_acl(Acl(), schemas=dict())
    assert 0 == len(acl)

    acl = Acl([Grant(privilege='ro', dbname=['db'], schema=None)])
    acl = manager.postprocess_acl(
        acl, schemas=dict(db=dict(
            public=['postgres', 'owner'],
            ns=['owner'],
        )),
    )

    # One grant per schema, per owner
    assert 3 == len(acl)


def test_postprocess_acl_bad_database():
    from ldap2pg.manager import SyncManager, Grant, Acl, UserError
    from ldap2pg.privilege import NspAcl
    from ldap2pg.utils import make_group_map

    privileges = dict(ro=NspAcl(name='ro', inspect='SQL'))
    manager = SyncManager(
        privileges=privileges, privilege_aliases=make_group_map(privileges),
    )

    acl = Acl([Grant('ro', ['inexistantdb'], None, 'alice')])
    schemas = dict(postgres=dict(public=['postgres']))

    with pytest.raises(UserError) as ei:
        manager.postprocess_acl(acl, schemas)
    assert 'inexistantdb' in str(ei.value)


def test_postprocess_acl_inexistant_privilege():
    from ldap2pg.manager import SyncManager, Acl, Grant, UserError

    manager = SyncManager()

    with pytest.raises(UserError):
        manager.postprocess_acl(
            acl=Acl([Grant('inexistant')]),
            schemas=dict(postgres=dict(public=['postgres'])),
        )


def test_inspect_ldap_roles(mocker):
    ql = mocker.patch('ldap2pg.manager.SyncManager.query_ldap')
    r = mocker.patch('ldap2pg.manager.SyncManager.process_ldap_entry')

    from ldap2pg.manager import SyncManager, Role

    ql.return_value = [mocker.Mock(name='entry')]
    r.side_effect = [
        {Role(name='alice', options=dict(SUPERUSER=True))},
        {Role(name='bob')},
    ]

    manager = SyncManager(
        ldapconn=mocker.Mock(),
    )

    # Minimal effective syncmap
    syncmap = [
        dict(roles=[]),
        dict(
            ldap=dict(base='ou=users,dc=tld', filter='*', attributes=['cn']),
            roles=[dict(), dict()],
        ),
    ]

    ldaproles, _ = manager.inspect_ldap(syncmap=syncmap)

    assert 2 is r.call_count, "sync did not iterate over each rules."

    assert 'alice' in ldaproles
    assert 'bob' in ldaproles


def test_inspect_roles_merge_duplicates(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    syncmap = [
        dict(roles=[
            dict(names=['group0']),
            dict(names=['group1']),
            dict(names=['bob'], parents=['group0']),
            dict(names=['bob'], parents=['group1']),
        ]),
    ]

    ldaproles, _ = manager.inspect_ldap(syncmap=syncmap)

    ldaproles = {r: r for r in ldaproles}
    assert 'group0' in ldaproles
    assert 'group1' in ldaproles
    assert 'bob' in ldaproles
    assert 3 == len(ldaproles)
    assert 2 == len(ldaproles['bob'].parents)


def test_inspect_roles_duplicate_differents_options(mocker):
    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager()

    syncmap = [dict(roles=[
        dict(names=['group0']),
        dict(names=['group1']),
        dict(names=['bob'], options=dict(LOGIN=True)),
        dict(names=['bob'], options=dict(LOGIN=False)),
    ])]

    with pytest.raises(UserError):
        manager.inspect_ldap(syncmap=syncmap)


def test_sync(mocker):
    from ldap2pg.manager import RoleOptions

    mod = 'ldap2pg.manager'
    mocker.patch(
        mod + '.RoleOptions.SUPPORTED_COLUMNS',
        RoleOptions.SUPPORTED_COLUMNS[:],
    )

    cls = mod + '.SyncManager'
    il = mocker.patch(cls + '.inspect_ldap', autospec=True)
    mocker.patch(cls + '.postprocess_acl', autospec=True)

    from ldap2pg.manager import SyncManager, UserError

    psql = mocker.Mock(name='psql')
    inspector = mocker.Mock(name='inspector')
    manager = SyncManager(psql=psql, inspector=inspector)

    inspector.fetch_me.return_value = ('postgres', False)
    inspector.roles_blacklist = ['pg_*']
    inspector.fetch_roles.return_value = (['postgres'], set(), set())
    pgroles = mocker.Mock(name='pgroles')
    # Simple diff with one query
    pgroles.diff.return_value = qry = [
        mocker.Mock(name='qry', args=(), message='hop')]
    inspector.filter_roles.return_value = set(), pgroles
    il.return_value = (mocker.Mock(name='ldaproles'), set())
    qry[0].expand.return_value = [qry[0]]
    inspector.fetch_schemas.return_value = dict(postgres=dict(ns=['owner']))
    inspector.fetch_grants.return_value = pgacl = mocker.Mock(name='pgacl')
    pgacl.diff.return_value = []

    # No privileges to sync, one query
    psql.dry = False
    psql.run_queries.return_value = 1
    count = manager.sync(syncmap=[])
    assert pgroles.diff.called is True
    assert pgacl.diff.called is False
    assert 1 == count

    # With privileges
    manager.privileges = dict(ro=mocker.Mock(name='ro'))
    count = manager.sync(syncmap=[])
    assert pgroles.diff.called is True
    assert pgacl.diff.called is True
    assert 2 == count

    # Dry run with roles and ACL
    manager.psql.dry = True
    manager.sync(syncmap=[])

    # Nothing to do
    psql.run_queries.return_value = 0
    count = manager.sync(syncmap=[])
    assert 0 == count

    # resolve_membership failure
    il.return_value[0].resolve_membership.side_effect = ValueError()
    with pytest.raises(UserError):
        manager.sync(syncmap=[])
