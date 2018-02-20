from __future__ import unicode_literals

from fnmatch import filter as fnfilter

import pytest


def test_generic_fetch(mocker):
    from ldap2pg.manager import psycopg2, SyncManager, UserError

    manager = SyncManager()
    psql = mocker.Mock(name='psql', side_effect=psycopg2.ProgrammingError())

    with pytest.raises(UserError):
        manager.pg_fetch(psql, 'POUET;')

    psql = mocker.Mock(name='psql', return_value=[('val0',), ('val1',)])
    rows = manager.pg_fetch(psql, 'POUET;', manager.row1)
    assert ['val0', 'val1'] == rows

    assert [] == manager.pg_fetch(None, None)

    assert [('val0',)] == manager.pg_fetch(None, ['val0'])
    assert [['val0']] == manager.pg_fetch(None, [['val0']])


def test_format_roles_inspect_sql(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()
    assert manager.format_roles_query() is None

    assert ['static'] == manager.format_roles_query(['static'])

    manager._roles_query = 'SELECT {options}'
    assert 'rolsuper' in manager.format_roles_query()


def test_process_roles_rows():
    from ldap2pg.manager import SyncManager

    manager = SyncManager(blacklist=['pg_*', 'postgres'])
    rows = [
        ('postgres', []),
        ('pg_signal_backend', []),
        ('dba', ['alice']),
        ('alice', []),
    ]
    roles = list(manager.process_pg_roles(rows))

    assert 2 == len(roles)
    assert 'dba' == roles[0].name
    assert 'alice' == roles[1].name


def test_process_acl_rows():
    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager(blacklist=['pg_*', 'postgres'])
    rows = [
        (None, 'postgres', True),
        (None, 'pg_signal_backend'),  # Old signature, fallback to True
        ('public', 'alice', True),
    ]

    items = list(manager.process_pg_acl_items('connect', 'postgres', rows))

    assert 1 == len(items)
    item = items[0]
    assert 'connect' == item.acl
    assert 'postgres' == item.dbname
    assert 'public' == item.schema
    assert 'alice' == item.role

    with pytest.raises(UserError):
        list(manager.process_pg_acl_items('acl', 'db', [('incomplete',)]))


def test_query_ldap(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search_s.return_value = [('dn=a', {}), ('dn=b', {})]

    entries = manager.query_ldap(
        base='ou=people,dc=global', filter='(objectClass=*)',
        scope=2, attributes=['cn'],
    )

    assert 2 == len(entries)


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
        options=dict(LOGIN=True),
    )
    roles = list(roles)

    assert 1 == len(roles)
    assert 'alice' in roles
    assert 'postgres' in roles[0].parents


def test_process_entry_user():
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = ('dn', {'cn': ['alice', 'bob']})

    roles = manager.process_ldap_entry(
        entry, name_attribute='cn',
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

    roles = manager.process_ldap_entry(entry, name_attribute='member.cn')
    roles = list(roles)
    names = {r.name for r in roles}

    assert 2 == len(roles)
    assert 'alice' in names
    assert 'bob' in names


def test_process_entry_membership(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = ('dn', {
        'cn': 'group',
        'member': ['cn=alice,dc=unit', 'cn=bob,dc=unit']})

    roles = manager.process_ldap_entry(
        entry, names=['group'], members_attribute='member.cn',
        parents_attribute='cn',
    )
    roles = list(roles)

    assert 1 == len(roles)
    role = roles[0]
    assert 'alice' in role.members
    assert 'bob' in role.members


def test_apply_role_rule_ko(mocker):
    gla = mocker.patch('ldap2pg.manager.get_attribute', autospec=True)

    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager()

    gla.side_effect = ValueError
    items = manager.apply_role_rules(
        entries=[('dn0',), ('dn1',)],
        rules=[dict(
            name_attribute='cn',
        )],
    )
    with pytest.raises(UserError):
        list(items)


def test_apply_grant_rule_ok(mocker):
    gla = mocker.patch('ldap2pg.manager.get_attribute', autospec=True)

    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    gla.side_effect = [['alice'], ['bob']]
    items = manager.apply_grant_rules(
        grant=[dict(
            acl='connect',
            database='postgres',
            schema='__any__',
            role_attribute='cn',
        )],
        entries=[None, None],
    )
    items = list(items)
    assert 2 == len(items)
    assert 'alice' == items[0].role
    assert 'postgres' == items[0].dbname
    # Ensure __any__ schema is mapped to None
    assert items[0].schema is None
    assert 'bob' == items[1].role


def test_apply_grant_rule_wrong_attr(mocker):
    gla = mocker.patch('ldap2pg.manager.get_attribute')

    from ldap2pg.manager import SyncManager, UserError

    gla.side_effect = ValueError('POUET')
    items = SyncManager().apply_grant_rules(
        grant=[dict(role_attribute='cn')],
        entries=[None, None],
    )
    with pytest.raises(UserError):
        list(items)


def test_apply_grant_rule_all_schema(mocker):
    gla = mocker.patch('ldap2pg.manager.get_attribute', autospec=True)

    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    gla.side_effect = [['alice']]
    items = manager.apply_grant_rules(
        grant=[dict(
            acl='connect',
            database='postgres',
            schema='__all__',
            role_attribute='cn',
        )],
        entries=[None],
    )
    items = list(items)
    assert 1 == len(items)
    assert 'alice' == items[0].role
    assert 'postgres' == items[0].dbname
    # Ensure __all__ schema is mapped to object
    assert items[0].schema != '__all__'


def test_apply_grant_rule_filter(mocker):
    from ldap2pg.manager import SyncManager

    items = SyncManager().apply_grant_rules(
        grant=[dict(
            acl='connect',
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
    gla = mocker.patch('ldap2pg.manager.get_attribute', autospec=True)

    from ldap2pg.manager import AclItem, SyncManager

    manager = SyncManager()

    gla.return_value = ['alice']
    items = list(manager.apply_grant_rules(
        grant=[dict(
            acl='connect',
            database='__all__', schema='__any__',
            role_attribute='cn',
        )],
        entries=[None],
    ))
    assert items[0].dbname is AclItem.ALL_DATABASES


def test_inspect_pg_acls(mocker):
    pa = mocker.patch(
        'ldap2pg.manager.SyncManager.process_pg_acl_items', autospec=True)

    from ldap2pg.manager import SyncManager, AclItem
    from ldap2pg.acl import NspAcl
    from ldap2pg.utils import make_group_map

    acl_dict = dict(
        noinspect=NspAcl(name='noinspect'),
        ro=NspAcl(name='ro', inspect='SQL'),
    )
    pa.return_value = [
        AclItem('ro', 'postgres', None, 'alice'),
        AclItem('ro', 'postgres', None, 'public'),
        AclItem('ro', 'postgres', None, 'unmanaged'),
        AclItem('ro', 'postgres', 'unmanaged', 'alice'),
    ]

    psql = mocker.MagicMock()
    psql.itersessions.return_value = [('postgres', psql)]
    manager = SyncManager(
        psql=psql, ldapconn=mocker.Mock(), acl_dict=acl_dict,
        acl_aliases=make_group_map(acl_dict)
    )
    manager._roles_query = managed_roles = ['alice']
    manager._schemas_query = ['public']
    manager._owners_query = ['postgres']
    syncmap = dict(db=dict(schema=[dict(roles=[], grant=dict(acl='ro'))]))

    schemas, owners, pgacls = manager.inspect_pg_acls(
        syncmap=syncmap, databases=['postgres'], roles=managed_roles)

    assert 2 == len(pgacls)
    assert 'postgres' in owners
    assert 'postgres' in schemas
    assert 'public' in schemas['postgres']
    grantees = [a.role for a in pgacls]
    assert 'public' in grantees
    assert 'alice' in grantees


def test_inspect_ldap_acls(mocker):
    la = mocker.patch(
        'ldap2pg.manager.SyncManager.apply_grant_rules', autospec=True)

    from ldap2pg.manager import SyncManager, AclItem
    from ldap2pg.acl import NspAcl
    from ldap2pg.utils import make_group_map

    acl_dict = dict(ro=NspAcl(name='ro'))
    la.return_value = [AclItem('ro', 'postgres', None, 'alice')]

    manager = SyncManager(
        psql=mocker.Mock(), ldapconn=mocker.Mock(), acl_dict=acl_dict,
        acl_aliases=make_group_map(acl_dict)
    )
    syncmap = [dict(roles=[], grant=dict(acl='ro'))]

    _, ldapacls = manager.inspect_ldap(syncmap=syncmap)

    assert 1 == len(ldapacls)


def test_postprocess_acls():
    from ldap2pg.manager import SyncManager, AclItem, AclSet
    from ldap2pg.acl import DefAcl

    manager = SyncManager(
        acl_dict=dict(ro=DefAcl(name='ro')),
        acl_aliases=dict(ro=['ro']),
    )

    # No owners
    ldapacls = manager.postprocess_acls(AclSet(), schemas=dict(), owners=[])
    assert 0 == len(ldapacls)

    ldapacls = AclSet([AclItem(acl='ro', dbname='db', schema=None)])
    ldapacls = manager.postprocess_acls(
        ldapacls, schemas=dict(db=['public', 'ns']),
        owners=['postgres', 'owner'],
    )

    # One item per schema, per owner
    assert 4 == len(ldapacls)


def test_postprocess_acls_bad_database():
    from ldap2pg.manager import SyncManager, AclItem, AclSet, UserError
    from ldap2pg.acl import NspAcl
    from ldap2pg.utils import make_group_map

    acl_dict = dict(ro=NspAcl(name='ro', inspect='SQL'))
    manager = SyncManager(
        acl_dict=acl_dict, acl_aliases=make_group_map(acl_dict)
    )

    ldapacls = AclSet([AclItem('ro', 'inexistantdb', None, 'alice')])
    schemas = dict(postgres=['public'])
    owners = ['postgres']

    with pytest.raises(UserError) as ei:
        manager.postprocess_acls(ldapacls, schemas, owners)
    assert 'inexistantdb' in str(ei.value)


def test_postprocess_acls_inexistant():
    from ldap2pg.manager import SyncManager, AclSet, AclItem, UserError

    manager = SyncManager()

    with pytest.raises(UserError):
        manager.postprocess_acls(
            ldapacls=AclSet([AclItem('inexistant')]),
            schemas=dict(postgres=['public']),
            owners=['postgres'],
        )


def test_inspect_pg_roles(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager(
        psql=mocker.MagicMock(),
        roles_query=[('spurious', [])],
    )
    manager._databases_query = ['postgres']

    databases, pgroles = manager.inspect_pg_roles()

    assert 'spurious' in pgroles
    assert 'postgres' in databases


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
        dict(names=['bob']),
    ])]

    with pytest.raises(UserError):
        manager.inspect_ldap(syncmap=syncmap)


def test_diff_roles():
    from ldap2pg.manager import SyncManager, Role, RoleSet

    m = SyncManager()

    pgroles = RoleSet([
        Role('drop-me'),
        Role('alter-me'),
        Role('nothing'),
    ])
    ldaproles = RoleSet([
        Role('alter-me', options=dict(LOGIN=True)),
        Role('nothing'),
        Role('create-me')
    ])
    queries = [q.args[0] for q in m.diff_roles(pgroles, ldaproles)]

    assert fnfilter(queries, 'ALTER ROLE "alter-me" WITH* LOGIN*;')
    assert fnfilter(queries, 'CREATE ROLE "create-me" *;')
    assert fnfilter(queries, '*DROP ROLE "drop-me";*')
    assert not fnfilter(queries, '*nothing*')


def test_diff_acls(mocker):
    from ldap2pg.acl import Acl, AclItem
    from ldap2pg.manager import SyncManager

    acl = Acl(name='connect', revoke='REVOKE {role}', grant='GRANT {role}')
    nogrant = Acl(name='nogrant', revoke='REVOKE')
    norvk = Acl(name='norvk', grant='GRANT')
    m = SyncManager(acl_dict={a.name: a for a in [acl, nogrant, norvk]})

    item0 = AclItem(acl=acl.name, dbname='backend', role='daniel')
    pgacls = set([
        item0,
        AclItem(acl=acl.name, dbname='backend', role='alice'),
        AclItem(acl=acl.name, dbname='backend', role='irrelevant', full=None),
        AclItem(acl=norvk.name, role='torevoke'),
    ])
    ldapacls = set([
        item0,
        AclItem(acl=acl.name, dbname='backend', role='david'),
        AclItem(acl=nogrant.name, role='togrant'),
    ])

    queries = [q.args[0] for q in m.diff_acls(pgacls, ldapacls)]

    assert not fnfilter(queries, 'REVOKE "daniel"*')
    assert fnfilter(queries, 'REVOKE "alice"*')
    assert fnfilter(queries, 'GRANT "david"*')


def test_run_queries_error(mocker):
    from ldap2pg.manager import SyncManager, UserError
    from ldap2pg.psql import Query

    psql = mocker.MagicMock(name='psql')
    cursor = psql.return_value.__enter__.return_value

    manager = SyncManager(psql=psql)
    queries = [
        Query('q0', Query.ALL_DATABASES, 'SQL 0'),
        Query('q1', 'postgres', 'SQL 1'),
    ]
    databases = ['postgres', 'template1']

    # Dry run
    manager.dry = True
    count = manager.run_queries(queries=queries, databases=databases)
    assert cursor.called is False
    assert 3 == count

    # Real mode
    cursor.side_effect = RuntimeError()
    manager.dry = False
    with pytest.raises(UserError):
        manager.run_queries(queries=queries, databases=databases)
    assert cursor.called is True


def test_sync(mocker):
    cls = 'ldap2pg.manager.SyncManager'
    ipa = mocker.patch(cls + '.inspect_pg_acls', autospec=True)
    ipr = mocker.patch(cls + '.inspect_pg_roles', autospec=True)
    il = mocker.patch(cls + '.inspect_ldap', autospec=True)
    mocker.patch(cls + '.postprocess_acls', autospec=True)
    dr = mocker.patch(cls + '.diff_roles', autospec=True)
    da = mocker.patch(cls + '.diff_acls', autospec=True)
    rq = mocker.patch(cls + '.run_queries', autospec=True)

    from ldap2pg.manager import SyncManager, UserError

    manager = SyncManager()

    ipr.return_value = (['postgres', 'template1'], set())
    il.return_value = (mocker.Mock(name='ldaproles'), set())
    # Simple diff with one query
    dr.return_value = qry = [mocker.Mock(name='qry', args=(), message='hop')]
    qry[0].expand.return_value = [qry[0]]
    ipa.return_value = (dict(postgres=['public']), [], set())
    da.return_value = []

    # No ACL to sync, one query
    rq.return_value = 1
    count = manager.sync(syncmap=[])
    assert dr.called is True
    assert da.called is False
    assert 1 == count

    # With ACLs
    manager.acl_dict = dict(ro=mocker.Mock(name='ro'))
    count = manager.sync(syncmap=[])
    assert dr.called is True
    assert da.called is True
    assert 2 == count

    # Nothing to do
    rq.return_value = 0
    count = manager.sync(syncmap=[])
    assert 0 == count

    il.return_value[0].resolve_membership.side_effect = ValueError()
    with pytest.raises(UserError):
        manager.sync(syncmap=[])
