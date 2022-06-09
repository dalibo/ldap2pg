import os

import pytest


def test_main(mocker):
    mocker.patch('ldap2pg.script.Configuration')
    s = mocker.patch('ldap2pg.script.synchronize', autospec=True)
    s.return_value = 0

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0 == ei.value.code


def test_bdb_quit(mocker):
    mocker.patch('ldap2pg.script.Configuration')
    s = mocker.patch('ldap2pg.script.synchronize')

    from ldap2pg.script import main, pdb

    s.side_effect = pdb.bdb.BdbQuit()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_unhandled_error(mocker):
    mocker.patch('ldap2pg.script.Configuration')
    s = mocker.patch('ldap2pg.script.synchronize')

    from ldap2pg.script import main

    s.side_effect = Exception()

    with pytest.raises(SystemExit) as ei:
        main()

    assert os.EX_SOFTWARE == ei.value.code


def test_user_error(mocker):
    mocker.patch('ldap2pg.script.Configuration')
    s = mocker.patch('ldap2pg.script.synchronize')

    from ldap2pg.script import main, UserError

    s.side_effect = UserError("Test message.", exit_code=0xCAFE)

    with pytest.raises(SystemExit) as ei:
        main()

    assert 0xCAFE == ei.value.code


def test_pdb(mocker):
    mocker.patch('ldap2pg.script.Configuration')
    mocker.patch('ldap2pg.script.os.environ', {'DEBUG': '1'})
    isatty = mocker.patch('ldap2pg.script.sys.stdout.isatty')
    isatty.return_value = True
    s = mocker.patch('ldap2pg.script.synchronize')
    s.side_effect = Exception()
    pm = mocker.patch('ldap2pg.script.pdb.post_mortem')

    from ldap2pg.script import main

    with pytest.raises(SystemExit) as ei:
        main()

    assert pm.called is True
    assert os.EX_SOFTWARE == ei.value.code


def test_synchronize(mocker):
    from ldap2pg.utils import Timer
    from ldap2pg.script import synchronize, Configuration

    Pooler = mocker.patch('ldap2pg.script.Pooler', autospec=True)
    clc = mocker.patch('ldap2pg.script.ldap.connect')
    SM = mocker.patch('ldap2pg.script.SyncManager', autospec=True)
    manager = SM.return_value
    manager.sync.return_value = 0
    manager.timer = Timer()

    config = mocker.MagicMock(name='config', spec=Configuration)
    # Dry run
    config.get.return_value = True
    synchronize(config=config)

    # Real mode
    config.get.return_value = False
    Pooler.getconn.return_value = mocker.MagicMock(name='conn')
    synchronize(config=config)

    assert clc.called is True
    assert manager.sync.called is True

    # No LDAP
    clc.reset_mock()
    config.has_ldapsearch.return_value = []
    synchronize(config=config)

    assert clc.called is False


def test_synchronize_ldapconn_error(mocker):
    from ldap2pg.script import synchronize, ConfigurationError, ldap

    mod = 'ldap2pg.script'
    mocker.patch(mod + '.Configuration', new=mocker.MagicMock)
    clc = mocker.patch(mod + '.ldap.connect')

    clc.side_effect = ldap.LDAPError("pouet")
    with pytest.raises(ConfigurationError):
        synchronize()


def test_synchronize_pgconn_error(mocker):
    from ldap2pg.script import synchronize, ConfigurationError, psycopg2

    mod = 'ldap2pg.script'
    mocker.patch(mod + '.Configuration', new=mocker.MagicMock)
    mocker.patch(mod + '.ldap.connect')
    Pooler = mocker.patch(mod + '.Pooler', autospec=True)

    pool = Pooler.return_value
    conn = pool.getconn.return_value
    conn.scalar.side_effect = psycopg2.OperationalError()

    with pytest.raises(ConfigurationError):
        synchronize()


def test_init_config_str():
    from ldap2pg.script import init_config

    config = init_config("""- role: myrole""", environ=dict(), argv=[])
    assert 1 == len(config['sync_map'])
    assert 'myrole' in str(config['sync_map'][0]['roles'][0].names[0])
