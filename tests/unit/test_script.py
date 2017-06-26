import os

import pytest


def test_main(mocker):
    mocker.patch('ldap2pg.script.logging.basicConfig', autospec=True)
    mocker.patch('ldap2pg.script.wrapped_main', autospec=True)

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

    assert os.EX_SOFTWARE == ei.value.code


def test_unhandled_error(mocker):
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main

    w.side_effect = Exception()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_user_error(mocker):
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main, UserError

    w.side_effect = UserError("Test message.", exit_code=0xCAFE)

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0xCAFE == ei.value.code


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
    assert os.EX_SOFTWARE == ei.value.code


def test_wrapped_main(mocker):
    c = mocker.patch('ldap2pg.script.Configuration', autospec=True)
    clc = mocker.patch('ldap2pg.script.create_ldap_connection')
    cpc = mocker.patch('ldap2pg.script.create_pg_connection')
    rm = mocker.patch('ldap2pg.script.RoleManager', autospec=True)

    from ldap2pg.script import wrapped_main

    wrapped_main()

    assert c.called is True
    assert clc.called is True
    assert cpc.called is True
    assert rm.called is True


def test_create_ldap(mocker):
    mocker.patch('ldap2pg.script.ldap3.Connection', autospec=True)
    from ldap2pg.script import create_ldap_connection

    conn = create_ldap_connection(
        host='ldap.company.com', port=None,
        bind='cn=admin,dc=company,dc=com', password='keepmesecret',
    )

    assert conn


def test_create_pgconn(mocker):
    mocker.patch('ldap2pg.script.psycopg2.connect', autospec=True)

    from ldap2pg.script import create_pg_connection

    conn = create_pg_connection(dsn="")

    assert conn
