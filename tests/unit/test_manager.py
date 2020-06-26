from __future__ import unicode_literals

import pytest


def test_query_ldap(mocker):
    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.return_value = [
        ('dn=a', {}),
        ('dn=b', {}),
        (None, {'ref': True}),
        (None, ['ldap://list_ref']),
    ]

    entries = manager.query_ldap(
        base='ou=people,dc=global', filter='(objectClass=*)',
        scope=2, joins={}, attributes=['cn'],
    )

    assert 2 == len(entries)

    manager.ldapconn.search_s.return_value = [('dn=a', {'a': b'\xbb'})]
    with pytest.raises(UserError):
        manager.query_ldap(
            base='ou=people,dc=global', filter='(objectClass=*)',
            scope=2, joins={}, attributes=['cn'],
        )


def test_query_ldap_joins_ok(mocker):
    from ldap2pg.manager import SyncManager

    search_result = [
        ('cn=A,ou=people,dc=global', {
            'cn': ['A'], 'member': ['cn=P,ou=people,dc=global']}),
        ('cn=B,ou=people,dc=global', {
            'cn': ['B'], 'member': ['cn=P,ou=people,dc=global']}),
    ]

    sub_search_result = [
        ('cn=P,ou=people,dc=global', {'sAMAccountName': ['P']}),
    ]

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.side_effect = [
            search_result, sub_search_result]

    entries = manager.query_ldap(
        base='ou=people,dc=global',
        filter='(objectClass=group)',
        scope=2,
        attributes=['cn', 'member'],
        joins={'member': dict(
            base='ou=people,dc=global',
            scope=2,
            filter='(objectClass=people)',
            attributes=['sAMAccountName'],
        )},
    )

    assert 2 == manager.ldapconn.search_s.call_count

    expected_entries = [
        ('cn=A,ou=people,dc=global',
         {
            'cn': ['A'],
            'dn': ['cn=A,ou=people,dc=global'],
            'member': ['cn=P,ou=people,dc=global'],
         },
         {
             'member': [('cn=P,ou=people,dc=global', {
                 'dn': ['cn=P,ou=people,dc=global'],
                 'samaccountname': ['P'],
             }, {})],
         }),
        ('cn=B,ou=people,dc=global',
         {
             'cn': ['B'],
             'dn': ['cn=B,ou=people,dc=global'],
             'member': ['cn=P,ou=people,dc=global'],
         },
         {
             'member': [('cn=P,ou=people,dc=global', {
                 'dn': ['cn=P,ou=people,dc=global'],
                 'samaccountname': ['P'],
             }, {})],
         }),
    ]

    assert expected_entries == entries


def test_query_ldap_joins_ignore_error(mocker):
    from ldap2pg.manager import SyncManager, LDAPError

    search_result = [
        ('cn=A,ou=people,dc=global', {
            'cn': ['A'],
            'member': ['cn=P,ou=people,dc=global']}),
    ]

    sub_search_result = LDAPError()

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.side_effect = [
            search_result, sub_search_result]

    entries = manager.query_ldap(
        base='ou=people,dc=global', filter='(objectClass=group)',
        scope=2, joins={'member': dict(
            base='ou=people,dc=global',
            scope=2,
            filter='(objectClass=people)',
            attributes=['sAMAccountName'],
        )},
        attributes=['cn', 'member'],
    )

    expected_entries = [
        ('cn=A,ou=people,dc=global', {
            'cn': ['A'],
            'dn': ['cn=A,ou=people,dc=global'],
            'member': ['cn=P,ou=people,dc=global'],
        }, {}),
    ]

    assert expected_entries == entries


def test_query_ldap_bad_filter(mocker):
    from ldap2pg.manager import SyncManager, LDAPError, UserError

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.side_effect = LDAPError()

    with pytest.raises(UserError):
        manager.query_ldap(
            base='dc=unit', filter='(broken',
            scope=2, joins={}, attributes=[],
        )

    assert manager.ldapconn.search_s.called is True


def test_inspect_ldap_unexpected_dn(mocker):
    ga = mocker.patch('ldap2pg.manager.get_attribute')
    ql = mocker.patch('ldap2pg.manager.SyncManager.query_ldap')

    from ldap2pg.manager import SyncManager, RDNError, UserError
    from ldap2pg.role import RoleRule

    manager = SyncManager()

    ga.side_effect = values = [
        ['member0_cn', RDNError(), 'member1_cn'],
    ]
    ql.return_value = [('dn0', {}, {})]

    list(manager.inspect_ldap([dict(
        ldap=dict(on_unexpected_dn='warn'),
        roles=[RoleRule(names=['{member.cn}'])],
    )]))

    ga.reset_mock()
    ga.side_effect = values

    list(manager.inspect_ldap([dict(
        ldap=dict(on_unexpected_dn='ignore'),
        roles=[RoleRule(names=['{member.cn}'])]
    )]))

    ga.reset_mock()
    ga.side_effect = values

    with pytest.raises(UserError):
        list(manager.inspect_ldap([dict(
            ldap=dict(),
            roles=[RoleRule(names=['{member.cn}'])],
        )]))


def test_inspect_ldap_missing_attribute(mocker):
    ql = mocker.patch('ldap2pg.manager.SyncManager.query_ldap')

    from ldap2pg.manager import SyncManager, UserError
    from ldap2pg.role import RoleRule

    manager = SyncManager()

    # Don't return member attribute.
    ql.return_value = [('dn0', {}, {})]

    with pytest.raises(UserError) as ei:
        list(manager.inspect_ldap([dict(
            ldap=dict(base='...'),
            # Request member attribute.
            roles=[RoleRule(names=['{member.cn}'])],
        )]))
    assert 'Missing attribute member' in str(ei.value)


def test_inspect_ldap_grants(mocker):
    from ldap2pg.manager import SyncManager
    from ldap2pg.privilege import Grant, NspAcl
    from ldap2pg.utils import make_group_map

    privileges = dict(ro=NspAcl(name='ro'))
    manager = SyncManager(
        psql=mocker.Mock(), ldapconn=mocker.Mock(), privileges=privileges,
        privilege_aliases=make_group_map(privileges),
        inspector=mocker.Mock(name='inspector'),
    )
    manager.inspector.roles_blacklist = ['blacklisted']
    rule = mocker.Mock(name='grant')
    rule.generate.return_value = [
        Grant('ro', 'postgres', None, 'alice'),
        Grant('ro', 'postgres', None, 'blacklisted'),
    ]
    syncmap = [dict(roles=[], grant=[rule])]

    _, grants = manager.inspect_ldap(syncmap=syncmap)

    assert 1 == len(grants)


def test_postprocess_grants():
    from ldap2pg.manager import SyncManager
    from ldap2pg.privilege import DefAcl, Grant, Acl

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
    from ldap2pg.manager import SyncManager, UserError
    from ldap2pg.privilege import NspAcl, Grant, Acl
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
    from ldap2pg.manager import SyncManager, UserError
    from ldap2pg.privilege import Acl, Grant

    manager = SyncManager()

    with pytest.raises(UserError):
        manager.postprocess_acl(
            acl=Acl([Grant('inexistant')]),
            schemas=dict(postgres=dict(public=['postgres'])),
        )


def test_inspect_ldap_roles(mocker):
    ql = mocker.patch('ldap2pg.manager.SyncManager.query_ldap')

    from ldap2pg.manager import SyncManager
    from ldap2pg.role import Role

    ql.return_value = [('dn', {}, {})]

    manager = SyncManager(
        ldapconn=mocker.Mock(),
        inspector=mocker.Mock(name='inspector'),
    )
    manager.inspector.roles_blacklist = ['blacklisted']

    rule0 = mocker.Mock(name='rule0', all_fields=[])
    rule0.generate.return_value = [Role('alice', options=dict(LOGIN=True))]
    rule1 = mocker.Mock(name='rule1', all_fields=[])
    rule1.generate.return_value = [Role('bob')]
    rule2 = mocker.Mock(name='rule2', all_fields=[])
    rule2.generate.return_value = [Role('blacklisted')]

    # Minimal effective syncmap
    syncmap = [
        dict(roles=[]),
        dict(
            ldap=dict(base='ou=users,dc=tld', filter='*', attributes=['cn']),
            roles=[rule0, rule1, rule2],
        ),
    ]

    ldaproles, _ = manager.inspect_ldap(syncmap=syncmap)

    assert 'alice' in ldaproles
    assert 'bob' in ldaproles


def test_inspect_roles_merge_duplicates(mocker):
    from ldap2pg.manager import SyncManager
    from ldap2pg.role import RoleRule

    manager = SyncManager()

    syncmap = [
        dict(roles=[
            RoleRule(names=['group0']),
            RoleRule(names=['group1']),
            RoleRule(names=['bob'], parents=['group0']),
            RoleRule(names=['bob'], parents=['group1']),
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
    from ldap2pg.role import RoleRule

    manager = SyncManager()

    syncmap = [dict(roles=[
        RoleRule(names=['group0']),
        RoleRule(names=['group1']),
        RoleRule(names=['bob'], options=dict(LOGIN=True)),
        RoleRule(names=['bob'], options=dict(LOGIN=False)),
    ])]

    with pytest.raises(UserError):
        manager.inspect_ldap(syncmap=syncmap)


def test_inspect_ldap_roles_comment_error(mocker):
    ql = mocker.patch('ldap2pg.manager.SyncManager.query_ldap')

    from ldap2pg.manager import SyncManager, UserError
    from ldap2pg.role import CommentError

    ql.return_value = [('dn', {}, {})]

    rule = mocker.Mock(name='rule', all_fields=[])
    rule.generate.side_effect = CommentError("message")
    rule.comment.formats = ['From {desc}']

    mapping = dict(
        ldap=dict(base='ou=users,dc=tld', filter='*', attributes=['cn']),
        roles=[rule],
    )

    manager = SyncManager()
    with pytest.raises(UserError):
        manager.inspect_ldap(syncmap=[mapping])


def test_empty_sync_map(mocker):
    from ldap2pg.manager import SyncManager, RoleSet

    manager = SyncManager(
        inspector=mocker.Mock(name='inspector'),
        psql=mocker.Mock(name='psql'),
    )
    manager.inspector.fetch_me.return_value = 'me', True
    manager.inspector.fetch_roles_blacklist.return_value = []
    manager.inspector.fetch_roles.return_value = [], RoleSet(), RoleSet()
    manager.inspector.filter_roles.return_value = RoleSet(), RoleSet()
    manager.psql.run_queries.return_value = 0

    manager.sync([])


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
    inspector.fetch_roles_blacklist.return_value = ['pg_*']
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
