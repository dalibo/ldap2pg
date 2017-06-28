from __future__ import unicode_literals

import pytest


def test_context_manager(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    with manager:
        assert manager.pgcursor


def test_fetch_rows(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.MagicMock()
    manager.pgcursor.__iter__.return_value = r = [mocker.Mock()]

    rows = manager.fetch_pg_roles()
    rows = list(rows)

    assert r == rows


def test_process_rows():
    from ldap2pg.manager import RoleManager

    manager = RoleManager(
        pgconn=None, ldapconn=None, blacklist=['pg_*', 'postgres'],
    )
    rows = [
        ('postgres',),
        ('pg_signal_backend',),
        ('alice',),
    ]
    roles = list(manager.process_pg_roles(rows))

    assert 1 == len(roles)
    assert 'alice' == roles[0].name


def test_fetch_entries(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())

    manager.ldapconn.entries = [
        mocker.Mock(cn=mocker.Mock(value='alice')),
        mocker.Mock(cn=mocker.Mock(value='bob')),
    ]
    entries = manager.query_ldap(
        base='ou=people,dc=global', filter='(objectClass=*)',
        attributes=['cn'],
    )

    assert 2 == len(entries)


def test_process_entry(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())

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


def test_psql(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.Mock()

    manager.psql('SELECT 1')

    assert manager.pgcursor.execute.called is True
    assert manager.pgconn.commit.called is True


def test_sync_bad_filter(mocker):
    mocker.patch('ldap2pg.manager.RoleManager.fetch_pg_roles')
    l = mocker.patch('ldap2pg.manager.RoleManager.query_ldap')
    r = mocker.patch('ldap2pg.manager.RoleManager.process_ldap_entry')

    from ldap2pg.manager import RoleManager, LDAPObjectClassError, UserError

    l.side_effect = LDAPObjectClassError()

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    map_ = [dict(ldap=dict(
        base='ou=people,dc=global', filter='(objectClass=*)',
        attributes=['cn'],
    ))]

    with pytest.raises(UserError):
        manager.sync(map_=map_)

    assert r.called is False


def test_sync_map_loop(mocker):
    p = mocker.patch('ldap2pg.manager.RoleManager.process_pg_roles')
    l = mocker.patch('ldap2pg.manager.RoleManager.query_ldap')
    r = mocker.patch('ldap2pg.manager.RoleManager.process_ldap_entry')
    psql = mocker.patch('ldap2pg.manager.RoleManager.psql')
    RoleSet = mocker.patch('ldap2pg.manager.RoleSet')

    from ldap2pg.manager import RoleManager, Role

    p.return_value = {Role(name='spurious')}
    l.return_value = [mocker.Mock(name='entry')]
    r.side_effect = [{Role(name='alice')}, {Role(name='bob')}]

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    # Minimal effective syncmap
    syncmap = [dict(
        ldap=dict(base='ou=users,dc=global', filter='*', attributes=['cn']),
        roles=[dict(), dict()],
    )]

    # No queries to run, we're just testing mapping loop
    RoleSet.return_value.diff.return_value = []

    manager.dry = False
    roles = manager.sync(map_=syncmap)

    assert 2 is r.call_count, "sync did not iterate over each rules."
    assert roles
    assert psql.called is False


def test_sync_query_loop(mocker):
    mocker.patch('ldap2pg.manager.RoleManager.process_pg_roles')
    psql = mocker.patch('ldap2pg.manager.RoleManager.psql')
    RoleSet = mocker.patch('ldap2pg.manager.RoleSet')

    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgconn.cursor.return_value.rowcount = -1

    # Simple diff with one query
    pgroles = RoleSet.return_value
    pgroles.diff.return_value = ["SELECT true;"]

    # Dry run
    manager.dry = True
    # No mapping, we're juste testing query loop
    roles = manager.sync(map_=[])

    assert psql.called is False

    # Real mode
    manager.dry = False
    roles = manager.sync(map_=[])
    assert roles
    assert psql.called is True
