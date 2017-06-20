import pytest


def test_main(mocker):
    environ = dict(
        LDAP_HOST='x',
        LDAP_BIND='x',
        LDAP_BASE='x',
        LDAP_PASSWORD='x',
    )
    mocker.patch('ldap2pg.script.os.environ', environ)
    mocker.patch('ldap2pg.script.logging.basicConfig')
    mocker.patch('ldap2pg.script.create_ldap_connection')
    mocker.patch('ldap2pg.script.create_pg_connection')
    mocker.patch('ldap2pg.script.RoleManager')

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0 == ei.value.code


def test_bdb_quit(mocker):
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main, pdb

    w.side_effect = pdb.bdb.BdbQuit()

    with pytest.raises(SystemExit) as ei:
        main()

    assert 1 == ei.value.code


def test_unhandled_error(mocker):
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main

    w.side_effect = Exception()

    with pytest.raises(SystemExit) as ei:
        main()

    assert 1 == ei.value.code


def test_pdb(mocker):
    mocker.patch('ldap2pg.script.os.environ', {'DEBUG': '1'})
    isatty = mocker.patch('ldap2pg.script.sys.stdout.isatty')
    isatty.return_value = True
    mocker.patch('ldap2pg.script.logging')
    w = mocker.patch('ldap2pg.script.wrapped_main')
    w.side_effect = Exception()
    pm = mocker.patch('ldap2pg.script.pdb.post_mortem')

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert pm.called is True
    assert 1 == ei.value.code


def test_create_ldap(mocker):
    mocker.patch('ldap2pg.script.ldap3.Connection', autospec=True)
    from ldap2pg.script import create_ldap_connection

    conn = create_ldap_connection(
        host='ldap.company.com',
        bind='cn=admin,dc=company,dc=com', password='keepmesecret',
    )

    assert conn


def test_create_pgconn(mocker):
    mocker.patch('ldap2pg.script.psycopg2.connect', autospec=True)

    from ldap2pg.script import create_pg_connection

    conn = create_pg_connection(dsn="")

    assert conn
