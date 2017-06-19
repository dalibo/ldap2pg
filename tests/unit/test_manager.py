def test_context_manager(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    with manager:
        assert manager.pgcursor


def test_fetch_existing_roles(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.Mock()

    manager.pgcursor.fetchall.return_value = [
        ('alice',),
        ('bob',),
    ]
    existing_roles = manager.fetch_pg_roles()

    assert {'alice', 'bob'} == existing_roles


def test_fetch_wanted_roles(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())

    manager.ldapconn.entries = [
        mocker.Mock(cn=mocker.Mock(value='alice')),
        mocker.Mock(cn=mocker.Mock(value='bob')),
    ]
    wanted_roles = manager.fetch_ldap_roles(
        base='ou=people,dc=global', query='(objectClass=*)',
    )

    assert {'alice', 'bob'} == wanted_roles


def test_create(mocker):
    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.pgcursor = mocker.Mock()
    manager.create('bob')

    assert manager.pgcursor.execute.called is True
    assert manager.pgconn.commit.called is True


def test_sync(mocker):
    p = mocker.patch('ldap2pg.manager.RoleManager.fetch_pg_roles')
    l = mocker.patch('ldap2pg.manager.RoleManager.fetch_ldap_roles')
    mocker.patch('ldap2pg.manager.RoleManager.create', autospec=True)

    p.return_value = set()
    l.return_value = {'alice', 'bob'}

    from ldap2pg.manager import RoleManager

    manager = RoleManager(pgconn=mocker.Mock(), ldapconn=mocker.Mock())
    manager.sync(base='ou=people,dc=global', query='(objectClass=*)')
