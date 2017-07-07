from __future__ import unicode_literals

from fnmatch import filter as fnfilter

import pytest


def test_fetch_databases(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()
    psql = mocker.Mock(name='psql')
    psql.return_value = mocker.MagicMock()
    psql.return_value.__iter__.return_value = [
        ('postgres',), ('template1',),
    ]

    rows = manager.fetch_database_list(psql)
    rows = list(rows)

    assert 2 == len(rows)
    assert 'postgres' in rows
    assert 'template1' in rows


def test_fetch_roles(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()
    psql = mocker.Mock(name='psql')
    psql.return_value = mocker.MagicMock()
    psql.return_value.__iter__.return_value = r = [mocker.Mock()]

    rows = manager.fetch_pg_roles(psql)
    rows = list(rows)

    assert r == rows


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
    from ldap2pg.manager import SyncManager

    manager = SyncManager(blacklist=['pg_*', 'postgres'])
    rows = [
        ('postgres', None, 'postgres'),
        ('template1', None, 'pg_signal_backend'),
        ('backend', 'public', 'alice'),
    ]

    items = list(manager.process_pg_acl_items('connect', rows))

    assert 1 == len(items)
    item = items[0]
    assert 'connect' == item.acl
    assert 'backend' == item.dbname
    assert 'public' == item.schema
    assert 'alice' == item.role


def test_query_ldap(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager(ldapconn=mocker.Mock())

    manager.ldapconn.entries = [
        mocker.Mock(cn=mocker.Mock(value='alice')),
        mocker.Mock(cn=mocker.Mock(value='bob')),
    ]
    entries = manager.query_ldap(
        base='ou=people,dc=global', filter='(objectClass=*)',
        attributes=['cn'],
    )

    assert 2 == len(entries)


def test_query_ldap_bad_filter(mocker):
    from ldap2pg.manager import SyncManager, LDAPExceptionError, UserError

    manager = SyncManager(ldapconn=mocker.Mock())
    manager.ldapconn.search.side_effect = LDAPExceptionError()

    with pytest.raises(UserError):
        manager.query_ldap(base='dc=unit', filter='(broken', attributes=[])

    assert manager.ldapconn.search.called is True


def test_process_entry_static():
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    roles = manager.process_ldap_entry(
        entry=None, names=['ALICE'], parents=['postgres'],
        options=dict(LOGIN=True),
    )
    roles = list(roles)

    assert 1 == len(roles)
    assert 'alice' in roles
    assert 'postgres' in roles[0].parents


def test_process_entry_user(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = mocker.Mock(entry_attributes_as_dict=dict(cn=['alice', 'bob']))

    roles = manager.process_ldap_entry(
        entry, name_attribute='cn',
        options=dict(LOGIN=True),
    )
    roles = list(roles)

    assert 2 == len(roles)
    assert 'alice' in roles
    assert 'bob' in roles
    assert roles[0].options['LOGIN'] is True


def test_process_entry_dn(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = mocker.Mock(
        entry_attributes_as_dict=dict(
            member=['cn=alice,dc=unit', 'cn=bob,dc=unit']),
    )

    roles = manager.process_ldap_entry(entry, name_attribute='member.cn')
    roles = list(roles)
    names = {r.name for r in roles}

    assert 2 == len(roles)
    assert 'alice' in names
    assert 'bob' in names


def test_process_entry_members(mocker):
    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    entry = mocker.Mock(
        entry_attributes_as_dict=dict(
            cn=['group'],
            member=['cn=alice,dc=unit', 'cn=bob,dc=unit'],
        ),
    )

    roles = manager.process_ldap_entry(
        entry, name_attribute='cn', members_attribute='member.cn',
    )
    roles = list(roles)

    assert 1 == len(roles)
    role = roles[0]
    assert 'alice' in role.members
    assert 'bob' in role.members


def test_apply_grant_rule_ok(mocker):
    gla = mocker.patch('ldap2pg.manager.get_ldap_attribute', autospec=True)

    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    gla.side_effect = [['alice'], ['bob']]
    items = manager.apply_grant_rules(
        grant=[dict(
            acl='connect',
            database='postgres',
            schema='__common__',
            role_attribute='cn',
        )],
        entries=[None, None],
    )
    items = list(items)
    assert 2 == len(items)
    assert 'alice' == items[0].role
    assert 'postgres' == items[0].dbname
    # Ensure __common__ schema is mapped to None
    assert items[0].schema is None
    assert 'bob' == items[1].role


def test_apply_grant_rule_filter(mocker):
    from ldap2pg.manager import SyncManager

    items = SyncManager().apply_grant_rules(
        grant=[dict(
            acl='connect',
            database='postgres',
            schema='__common__',
            role_match='*_r',
            roles=['alice_r', 'bob_rw'],
        )],
        entries=[None],
    )
    items = list(items)
    assert 1 == len(items)
    assert 'alice_r' == items[0].role


def test_apply_grant_rule_nodb(mocker):
    gla = mocker.patch('ldap2pg.manager.get_ldap_attribute', autospec=True)

    from ldap2pg.manager import SyncManager

    manager = SyncManager()

    gla.return_value = ['alice']
    with pytest.raises(ValueError):
        list(manager.apply_grant_rules(
            grant=[dict(
                acl='connect',
                database='__common__', schema='__common__',
                role_attribute='cn',
            )],
            entries=[None],
        ))


def test_inspect_acls(mocker):
    mod = 'ldap2pg.manager.'
    psql = mocker.MagicMock()
    psql.itersessions.return_value = [('postgres', psql)]

    dbl = mocker.patch(mod + 'SyncManager.fetch_database_list', autospec=True)
    dbl.return_value = ['postgres']
    mocker.patch(mod + 'SyncManager.process_pg_roles', autospec=True)
    pa = mocker.patch(mod + 'SyncManager.process_pg_acl_items', autospec=True)
    la = mocker.patch(mod + 'SyncManager.apply_grant_rules', autospec=True)

    from ldap2pg.manager import SyncManager, AclItem
    from ldap2pg.acl import Acl

    acl_dict = dict(
        noinspect=Acl(name='noinspect'),
        ro=Acl(name='ro', inspect='SQL'),
    )
    pa.return_value = [AclItem('ro', 'postgres', None, 'alice')]
    la.return_value = [AclItem('ro', 'postgres', None, 'alice')]

    manager = SyncManager(psql=psql, ldapconn=mocker.Mock(), acl_dict=acl_dict)
    syncmap = dict(db=dict(schema=[dict(roles=[], grant=dict(acl='ro'))]))

    databases, _, pgacls, _, ldapacls = manager.inspect(syncmap=syncmap)

    assert 1 == len(pgacls)
    assert 1 == len(ldapacls)


def test_inspect_roles(mocker):
    p = mocker.patch('ldap2pg.manager.SyncManager.process_pg_roles')
    l = mocker.patch('ldap2pg.manager.SyncManager.query_ldap')
    r = mocker.patch('ldap2pg.manager.SyncManager.process_ldap_entry')
    psql = mocker.MagicMock()

    from ldap2pg.manager import SyncManager, Role

    p.return_value = {Role(name='spurious')}
    l.return_value = [mocker.Mock(name='entry')]
    r.side_effect = [{Role(name='alice')}, {Role(name='bob')}]

    manager = SyncManager(psql=psql, ldapconn=mocker.Mock())
    # Minimal effective syncmap
    syncmap = dict(db=dict(s=[
        dict(roles=[]),
        dict(
            ldap=dict(base='ou=users,dc=tld', filter='*', attributes=['cn']),
            roles=[dict(), dict()],
        ),
    ]))

    manager.inspect(syncmap=syncmap)

    assert 2 is r.call_count, "sync did not iterate over each rules."


def test_diff_roles(mocker):
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
    queries = [q.args[0] for q in m.diff(pgroles, set(), ldaproles, set())]

    assert fnfilter(queries, "ALTER ROLE alter-me WITH* LOGIN*;")
    assert fnfilter(queries, "CREATE ROLE create-me *;")
    assert fnfilter(queries, '*DROP ROLE drop-me;*')
    assert not fnfilter(queries, '*nothing*')


def test_diff_acls(mocker):
    from ldap2pg.acl import Acl, AclItem
    from ldap2pg.manager import SyncManager

    acl = Acl(name='connect', revoke='REVOKE %(role)s', grant='GRANT %(role)s')
    noquery = Acl(name='noquery')
    m = SyncManager(acl_dict={acl.name: acl, noquery.name: noquery})

    item0 = AclItem(acl=acl.name, dbname='backend', role='daniel')
    pgacls = set([
        item0,
        AclItem(acl=acl.name, dbname='backend', role='alice'),
        AclItem(acl=noquery.name, role='torevoke'),
    ])
    ldapacls = set([
        item0,
        AclItem(acl=acl.name, dbname='backend', role='david'),
        AclItem(acl=noquery.name, role='togrant'),
    ])

    queries = [q.args[0] for q in m.diff(pgacls=pgacls, ldapacls=ldapacls)]

    assert not fnfilter(queries, "REVOKE daniel*")
    assert fnfilter(queries, "REVOKE alice*")
    assert fnfilter(queries, "GRANT david*")


def test_sync(mocker):
    diff = mocker.patch('ldap2pg.manager.SyncManager.diff')

    from ldap2pg.manager import SyncManager

    psql = mocker.MagicMock()
    cursor = psql.return_value.__enter__.return_value

    manager = SyncManager(psql=psql)

    # Simple diff with one query
    diff.return_value = qry = [mocker.Mock(name='qry', args=())]
    qry[0].expand.return_value = [qry[0]]

    sync_kw = dict(
        databases=['postgres', 'template1'],
        pgroles=set(), pgacls=set(), ldaproles=set(), ldapacls=set(),
    )

    # Dry run
    manager.dry = True
    # No mapping, we're just testing query loop
    manager.sync(**sync_kw)
    assert cursor.called is False

    # Real mode
    manager.dry = False
    manager.sync(**sync_kw)
    assert cursor.called is True

    # Nothing to do
    diff.return_value = []
    manager.dry = False
    manager.sync(**sync_kw)
    assert cursor.called is True
