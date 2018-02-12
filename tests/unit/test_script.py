import os

import pytest


def test_main(mocker):
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
    wm = mocker.patch('ldap2pg.script.wrapped_main', autospec=True)
    wm.return_value = 0

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0 == ei.value.code


def test_bdb_quit(mocker):
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main, pdb

    w.side_effect = pdb.bdb.BdbQuit()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_unhandled_error(mocker):
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main

    w.side_effect = Exception()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_user_error(mocker):
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
    w = mocker.patch('ldap2pg.script.wrapped_main')

    from ldap2pg.script import main, UserError

    w.side_effect = UserError("Test message.", exit_code=0xCAFE)

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0xCAFE == ei.value.code


def test_pdb(mocker):
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
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
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
    PSQL = mocker.patch('ldap2pg.script.PSQL', autospec=True)
    clc = mocker.patch('ldap2pg.script.ldap.connect')
    SM = mocker.patch('ldap2pg.script.SyncManager', autospec=True)
    manager = SM.return_value
    manager.sync.return_value = 0

    from ldap2pg.script import wrapped_main

    config = mocker.MagicMock(name='config')
    # Dry run
    config.get.return_value = True
    wrapped_main(config=config)

    # Real mode
    config.get.return_value = False
    PSQL.return_value.return_value = mocker.MagicMock(name='psql')
    wrapped_main(config=config)

    assert clc.called is True
    assert manager.sync.called is True

    # No LDAP
    clc.reset_mock()
    config.has_ldap_query.return_value = []
    wrapped_main(config=config)

    assert clc.called is False


def test_conn_errors(mocker):
    mocker.patch('ldap2pg.script.dictConfig', autospec=True)
    mocker.patch('ldap2pg.script.Configuration', autospec=True)
    mocker.patch('ldap2pg.script.SyncManager', autospec=True)
    clc = mocker.patch('ldap2pg.script.ldap.connect')
    PSQL = mocker.patch('ldap2pg.script.PSQL', autospec=True)

    from ldap2pg.script import (
        wrapped_main, ConfigurationError,
        ldap, psycopg2,
    )

    clc.side_effect = ldap.LDAPError("pouet")
    with pytest.raises(ConfigurationError):
        wrapped_main()
    clc.side_effect = None

    psql = PSQL.return_value
    psql.return_value = mocker.MagicMock()
    psql_ = psql.return_value.__enter__.return_value
    psql_.side_effect = psycopg2.OperationalError()
    with pytest.raises(ConfigurationError):
        wrapped_main()
