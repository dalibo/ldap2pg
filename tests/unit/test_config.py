from __future__ import unicode_literals

import pytest


class MockArgs(dict):
    def __getattr__(self, name):
        try:
            return self[name]
        except KeyError:
            raise AttributeError(name)


def test_multiline_formatter():
    import logging
    from ldap2pg.config import MultilineFormatter

    formatter = MultilineFormatter("prefix: %(message)s")

    base_record = dict(
        name='pouet', level=logging.DEBUG, fn="(unknown file)", lno=0, args=(),
        exc_info=None,
    )
    record = logging.makeLogRecord(dict(base_record, msg="single line"))
    payload = formatter.format(record)
    assert "prefix: single line" == payload

    record = logging.makeLogRecord(dict(base_record, msg="Uno\nDos\nTres"))

    payload = formatter.format(record)
    wanted = """\
    prefix: Uno
    prefix: Dos
    prefix: Tres\
    """.replace('    ', '')

    assert wanted == payload


def test_color_handler():
    import logging
    from ldap2pg.config import ColoredStreamHandler

    handler = ColoredStreamHandler()
    record = logging.makeLogRecord(dict(
        name='pouet', level=logging.DEBUG, fn="(unknown file)", msg="Message",
        lno=0, args=(), exc_info=None,
    ))
    payload = handler.format(record)
    assert "\033[0" in payload


def test_logging_config():
    from ldap2pg.config import Configuration

    config = Configuration()

    config['verbose'] = True
    dict_ = config.logging_dict()
    assert 'DEBUG' == dict_['loggers']['ldap2pg']['level']

    config['verbose'] = False
    dict_ = config.logging_dict()
    assert 'INFO' == dict_['loggers']['ldap2pg']['level']


def test_mapping():
    from ldap2pg.config import Mapping

    m = Mapping('my:option', env=None)
    assert 'my_option' == m.arg
    assert 'my:option' in repr(m)

    # Fallback to default
    v = m.process(default='defval', file_config=dict(), environ=dict())
    assert 'defval' == v

    # Read file
    v = m.process(
        default='defval',
        file_config=dict(my=dict(option='fileval')),
        environ=dict(),
    )
    assert 'fileval' == v

    # Ignore env
    v = m.process(
        default='defval',
        file_config=dict(my=dict(option='fileval')),
        environ=dict(MY_OPTION=b'envval'),
    )
    assert 'fileval' == v

    m = Mapping('my:option')
    assert 'MY_OPTION' in m.env
    assert 'MYOPTION' in m.env

    # Prefer env over file
    v = m.process(
        default='defval',
        file_config=dict(my=dict(option='fileval')),
        environ=dict(MY_OPTION=b'envval'),
    )
    assert 'envval' == v

    # Prefer argv over env
    v = m.process(
        default='defval',
        file_config=dict(my=dict(option='fileval')),
        environ=dict(MY_OPTION='envval'),
        args=MockArgs(my_option='argval')
    )
    assert 'argval' == v


def test_mapping_security():
    from ldap2pg.config import Mapping

    m = Mapping('ldap:password', secret=True)
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
        environ=dict(LDAP_PASSWORD=b'fromenv'),
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


def test_processor():
    from ldap2pg.config import Mapping

    m = Mapping('dry', processor=bool)
    v = m.process(default=True, file_config=dict(dry=0), environ=dict())

    assert v is False


def test_find_filename_default(mocker):
    stat = mocker.patch('ldap2pg.config.stat')

    from ldap2pg.config import Configuration, NoConfigurationError

    config = Configuration()

    def mk_oserror(errno=None):
        e = OSError()
        e.errno = errno
        return e

    # Search default path
    stat.side_effect = [
        mk_oserror(),
        mk_oserror(13),
        mocker.Mock(st_mode=0o600),
    ]
    filename, mode = config.find_filename(environ=dict())
    assert config._file_candidates[2] == filename
    assert 0o600 == mode

    # No files at all
    stat.side_effect = OSError()
    with pytest.raises(NoConfigurationError):
        config.find_filename(environ=dict())


def test_find_filename_custom(mocker):
    stat = mocker.patch('ldap2pg.config.stat')

    from ldap2pg.config import Configuration, UserError

    config = Configuration()

    # Read from env var LDAP2PG_CONFIG
    stat.reset_mock()
    stat.side_effect = [
        OSError(),
        AssertionError("Not reached."),
    ]
    with pytest.raises(UserError):
        config.find_filename(environ=dict(LDAP2PG_CONFIG=b'my.yml'))

    # Read from args
    stat.reset_mock()
    stat.side_effect = [
        mocker.Mock(st_mode=0o600),
        AssertionError("Not reached."),
    ]
    filename, mode = config.find_filename(
        environ=dict(LDAP2PG_CONFIG=b'env.yml'),
        args=MockArgs(config='argv.yml'),
    )

    assert filename.endswith('argv.yml')


def test_find_filename_stdin():
    from ldap2pg.config import Configuration

    config = Configuration()

    filename, mode = config.find_filename(
        environ=dict(LDAP2PG_CONFIG=b'-'),
    )

    assert '-' == filename
    assert 0o400 == mode


def test_merge():
    from ldap2pg.config import Configuration

    # Noop
    config = Configuration()
    config.merge(file_config={}, environ={})

    minimal_config = dict(sync_map=[])
    config.merge(
        file_config=minimal_config,
        environ=dict(),
    )
    config.merge(
        file_config=minimal_config,
        environ=dict(LDAPPASSWORD=b'envpass', PGDSN=b'envdsn'),
    )
    assert 'envpass' == config['ldap']['password']
    assert 'envdsn' == config['postgres']['dsn']


def test_security():
    from ldap2pg.config import Configuration

    config = Configuration()

    minimal_config = dict(sync_map=[])
    with pytest.raises(ValueError):
        config.merge(environ=dict(), file_config=dict(
            minimal_config,
            ldap=dict(password='unsecure'),
        ))

    with pytest.raises(ValueError):
        # Refuse world readable postgres URI with password
        config.merge(environ=dict(), file_config=dict(
            minimal_config,
            postgres=dict(dsn='password=unsecure'),
        ))

    with pytest.raises(ValueError):
        # Refuse world readable postgres URI with password
        config.merge(environ=dict(), file_config=dict(
            minimal_config,
            postgres=dict(dsn='postgres://u:unsecure@h'),
        ))

    config.merge(environ=dict(), file_config=dict(
        minimal_config,
        postgres=dict(dsn='postgres://u@h'),
    ))


def test_read_yml():
    from io import StringIO

    from ldap2pg.config import Configuration, ConfigurationError

    config = Configuration()

    fo = StringIO("- role: alice")
    payload = config.read(fo, 'memory', mode=0o0)
    assert 'sync_map' in payload

    fo = StringIO("entry: value")
    payload = config.read(fo, 'memory', mode=0o644)
    assert 'entry' in payload
    assert payload['world_readable'] is True

    # Accept empty file (e.g. /dev/null)
    fo = StringIO("")
    payload = config.read(fo, 'memory', mode=0o600)
    assert payload['world_readable'] is False

    with pytest.raises(ConfigurationError):
        fo = StringIO("bad_value")
        payload = config.read(fo, 'memory', mode=0o600)

    with pytest.raises(ConfigurationError):
        fo = StringIO("bad: { yaml ] *&")
        payload = config.read(fo, 'memory', mode=0o600)


def test_load_badfiles(mocker):
    environ = dict()
    mocker.patch('ldap2pg.config.os.environ', environ)
    ff = mocker.patch('ldap2pg.config.Configuration.find_filename')
    merge = mocker.patch('ldap2pg.config.Configuration.merge')

    from ldap2pg.config import (
        Configuration,
        ConfigurationError,
        NoConfigurationError,
        UserError,
    )

    config = Configuration()

    # No file specified
    ff.side_effect = NoConfigurationError()
    config.load(argv=[])

    ff.side_effect = None
    # Invalid file
    ff.return_value = ['filename.yml', 0o0]
    merge.side_effect = ValueError()
    o = mocker.patch('ldap2pg.config.open', mocker.mock_open(), create=True)
    with pytest.raises(ConfigurationError):
        config.load(argv=[])

    # Not readable.
    o.side_effect = OSError("failed to open")
    with pytest.raises(UserError):
        config.load(argv=[])


def test_load_stdin(mocker):
    environ = dict()
    mocker.patch('ldap2pg.config.os.environ', environ)
    ff = mocker.patch('ldap2pg.config.Configuration.find_filename')
    mocker.patch('ldap2pg.config.open', create=True)
    read = mocker.patch('ldap2pg.config.Configuration.read')

    from ldap2pg.config import Configuration

    config = Configuration()

    ff.return_value = ['-', 0o400]
    read.return_value = dict(sync_map=[dict(role='alice')])

    config.load(argv=[])

    maplist = config['sync_map']['__all__']['__any__']
    assert 1 == len(maplist)


def test_load_file(mocker):
    environ = dict()
    mocker.patch('ldap2pg.config.os.environ', environ)
    ff = mocker.patch('ldap2pg.config.Configuration.find_filename')
    mocker.patch('ldap2pg.config.open', create=True)
    read = mocker.patch('ldap2pg.config.Configuration.read')

    from ldap2pg.config import Configuration

    config = Configuration()

    ff.return_value = ['filename.yml', 0o0]
    read.return_value = dict(sync_map=[dict(role='alice')])
    # send one env var for LDAP bind
    environ.update(dict(LDAPPASSWORD=b'envpass'))

    config.load(argv=['--verbose'])

    assert 'envpass' == config['ldap']['password']
    maplist = config['sync_map']['__all__']['__any__']
    assert 1 == len(maplist)
    assert config['verbose'] is True


def test_show_versions(mocker):
    from ldap2pg.config import Configuration

    config = Configuration()
    with pytest.raises(SystemExit):
        config.load(argv=['--version'])
