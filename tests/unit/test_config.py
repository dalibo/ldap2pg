import pytest


def test_mapping():
    from ldap2pg.config import Mapping

    m = Mapping('ldap:password', secret=True)
    assert 'LDAP_PASSWORD' == m.env

    # Nothing in either file or env -> default
    v = m.process(default='DEFAULT', file_config=dict(), environ=dict())
    assert 'DEFAULT' == v

    with pytest.raises(ValueError):
        # Something in file but it's not secure
        m.process(
            default='DEFAULT',
            file_config=dict(
                world_readable=True,
                ldap=dict(password='unsecure'),
            ),
            environ=dict(),
        )

    # File is unsecure but env var overrides value and error.
    v = m.process(
        default='DEFAULT',
        file_config=dict(ldap=dict(password='unsecure')),
        environ=dict(LDAP_PASSWORD='fromenv'),
    )
    assert 'fromenv' == v

    # File is secure, use it.
    v = m.process(
        default='DEFAULT',
        file_config=dict(world_readable=False, ldap=dict(password='53cUr3!')),
        environ=dict(),
    )
    assert '53cUr3!' == v

    m = Mapping('postgres:dsn', secret="password=")
    with pytest.raises(ValueError):
        # Something in file but it's not secure
        m.process(
            default='DEFAULT',
            file_config=dict(
                world_readable=True,
                postgres=dict(dsn='password=unsecure'),
            ),
            environ=dict(),
        )


def test_find_filename(mocker):
    stat = mocker.patch('ldap2pg.config.stat')

    from ldap2pg.config import Configuration

    config = Configuration()

    # Search default path
    stat.side_effect = [
        FileNotFoundError(),
        PermissionError(),
        mocker.Mock(st_mode=0o600),
    ]
    filename, mode = config.find_filename(environ=dict())
    assert config._file_candidates[2] == filename
    assert 0o600 == mode

    # Read from env var LDAP2PG_CONFIG
    stat.reset_mock()
    stat.side_effect = [
        PermissionError(),
        AssertionError("Not reached."),
    ]
    with pytest.raises(FileNotFoundError):
        config.find_filename(environ=dict(LDAP2PG_CONFIG='my.yml'))


def test_merge_and_mappings():
    from ldap2pg.config import Configuration

    # Noop
    config = Configuration()
    config.merge(file_config={}, environ={})

    config.merge(
        file_config=dict(ldap=dict(host='confighost')),
        environ=dict(LDAP_PASSWORD='envpass', PGDSN='envdsn'),
    )
    assert 'confighost' == config['ldap']['host']
    assert 'envpass' == config['ldap']['password']
    assert 'envdsn' == config['postgres']['dsn']

    with pytest.raises(ValueError):
        config.merge(
            file_config=dict(ldap=dict(password='unsecure')),
            environ=dict(),
        )

    with pytest.raises(ValueError):
        # Refuse world readable postgres URI with password
        config.merge(
            file_config=dict(postgres=dict(dsn='password=unsecure')),
            environ=dict(),
        )

    with pytest.raises(ValueError):
        # Refuse world readable postgres URI with password
        config.merge(
            file_config=dict(postgres=dict(dsn='postgres://u:unsecure@h')),
            environ=dict(),
        )

    config.merge(
        file_config=dict(postgres=dict(dsn='postgres://u@h')),
        environ=dict(),
    )


def test_read_yml():
    from io import StringIO

    from ldap2pg.config import Configuration

    config = Configuration()

    # Deny list file
    fo = StringIO("- listentry")
    with pytest.raises(ValueError):
        config.read(fo, mode=0o0)

    fo = StringIO("entry: value")
    payload = config.read(fo, mode=0o644)
    assert 'entry' in payload
    assert payload['world_readable'] is True

    # Accept empty file (e.g. /dev/null)
    fo = StringIO("")
    payload = config.read(fo, mode=0o600)
    assert payload['world_readable'] is False


def test_load(mocker):
    environ = dict()
    mocker.patch('ldap2pg.config.os.environ', environ)
    ff = mocker.patch('ldap2pg.config.Configuration.find_filename')
    read = mocker.patch('ldap2pg.config.Configuration.read')
    mocker.patch('ldap2pg.config.open')

    from ldap2pg.config import Configuration

    config = Configuration()

    ff.side_effect = FileNotFoundError()
    # Noop: just use defaults
    config.load()

    ff.side_effect = None
    # Find `filename.yml`
    ff.return_value = ['filename.yml', 0o0]
    # ...containing LDAP host
    read.return_value = dict(ldap=dict(host='cfghost'))
    # send one env var for LDAP bind
    environ.update(dict(LDAP_BIND='envbind'))

    config.load()

    assert 'cfghost' == config['ldap']['host']
    assert 'envbind' == config['ldap']['bind']
