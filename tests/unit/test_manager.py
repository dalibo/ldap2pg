from __future__ import unicode_literals

import pytest


def test_role():
    from ldap2pg.manager import Role

    role = Role(name='toto')

    assert 'toto' == role.name
    assert 'toto' == str(role)
    assert 'toto' in repr(role)


def test_context_manager(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    with manager:
        assert manager.pgcursor


def test_blacklist():
    from ldap2pg.manager import RoleManager

    manager = RoleManager(
        pgconn=None, ldapconn=None, blacklist=['pg_*', 'postgres'],
    )
    roles = ['postgres', 'pg_signal_backend', 'alice', 'bob']
    filtered = list(manager.blacklist(roles))
    assert ['alice', 'bob'] == filtered


def test_fetch_existing_roles(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.Mock()

    manager.pgcursor.fetchall.return_value = [
        ('alice',),
        ('bob',),
    ]
    existing_roles = manager.fetch_pg_roles()

    assert {'alice', 'bob'} == {r.name for r in existing_roles}


def test_fetch_wanted_roles(mocker):
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

    roles = manager.process_ldap_entry(entry, name_attribute='cn')
    roles = list(roles)

    assert 2 == len(roles)
    assert 'alice' in roles
    assert 'bob' in roles

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


def test_create(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.Mock()

    manager.dry = True
    manager.create('bob')

    assert manager.pgcursor.execute.called is False
    assert manager.pgconn.commit.called is False

    manager.dry = False
    manager.create('bob')

    assert manager.pgcursor.execute.called is True
    assert manager.pgconn.commit.called is True


def test_drop(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.Mock()

    manager.dry = True
    manager.drop('alice')

    assert manager.pgcursor.execute.called is False
    assert manager.pgconn.commit.called is False

    manager.dry = False
    manager.drop('alice')

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


def test_sync(mocker):
    p = mocker.patch('ldap2pg.manager.RoleManager.fetch_pg_roles')
    l = mocker.patch('ldap2pg.manager.RoleManager.query_ldap')
    r = mocker.patch('ldap2pg.manager.RoleManager.process_ldap_entry')

    p.return_value = {'spurious'}
    l.return_value = [mocker.Mock(name='entry')]
    r.side_effect = [{'alice'}, {'bob'}]

    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    map_ = [dict(
        ldap=dict(
            base='ou=people,dc=global', filter='(objectClass=*)',
            attributes=['cn'],
        ),
        roles=[
            dict(name_attribute='cn'),
            dict(name_attribute='pouet'),
        ],
    )]

    roles = manager.sync(map_=map_)

    assert 2 is r.call_count, "sync did not iterate over each rules."
    assert 'alice' in roles
    assert 'bob' in roles
