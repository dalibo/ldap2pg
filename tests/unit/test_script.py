import os

import pytest


def test_main(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    mocker.patch('ldap2pg.script.wrapped_main', autospec=True)

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0 == ei.value.code


def test_bdb_quit(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main, pdb

    w.side_effect = pdb.bdb.BdbQuit()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_unhandled_error(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main

    w.side_effect = Exception()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_user_error(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main, UserError

    w.side_effect = UserError("Test message.", exit_code=0xCAFE)

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0xCAFE == ei.value.code


def test_pdb(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    mocker.patch('ldap2pg.script.os.environ', {'DEBUG': '1'})
    isatty = mocker.patch('ldap2pg.script.sys.stdout.isatty')
    isatty.return_value = True
    w = mocker.patch('ldap2pg.script.wrapped_main')
    w.side_effect = Exception()
    pm = mocker.patch('ldap2pg.script.pdb.post_mortem')

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert pm.called is True
    assert os.EX_SOFTWARE == ei.value.code


def test_wrapped_main(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    clc = mocker.patch('ldap2pg.script.create_ldap_connection')
    RM = mocker.patch('ldap2pg.script.SyncManager', autospec=True)
    rm = RM.return_value
    rm.inspect.return_value = [mocker.Mock()] * 5

    from ldap2pg.script import wrapped_main

    config = mocker.MagicMock(name='config')
    config.get.return_value = True
    wrapped_main(config=config)

    config.get.return_value = False
    wrapped_main(config=config)

    assert clc.called is True
    assert rm.inspect.called is True
    assert rm.sync.called is True


def test_conn_errors(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    mocker.patch('ldap2pg.script.Configuration', autospec=True)
    SyncManager = mocker.patch('ldap2pg.script.SyncManager', autospec=True)
    SyncManager.return_value.inspect.return_value = [mocker.Mock()] * 3
    clc = mocker.patch('ldap2pg.script.create_ldap_connection')

    from ldap2pg.script import (
        wrapped_main, ConfigurationError,
        ldap3, psycopg2,
    )

    clc.side_effect = ldap3.core.exceptions.LDAPExceptionError("pouet")
    with pytest.raises(ConfigurationError):
        wrapped_main()

    clc.side_effect = None
    manager = SyncManager.return_value
    manager.inspect.side_effect = psycopg2.OperationalError()
    with pytest.raises(ConfigurationError):
        wrapped_main()


def test_create_ldap(mocker):
    mocker.patch('ldap2pg.script.logging.config.dictConfig', autospec=True)
    mocker.patch('ldap2pg.script.ldap3.Connection', autospec=True)
    from ldap2pg.script import create_ldap_connection

    conn = create_ldap_connection(
        host='ldap.company.com', port=None,
        bind='cn=admin,dc=company,dc=com', password='keepmesecret',
    )

    assert conn
