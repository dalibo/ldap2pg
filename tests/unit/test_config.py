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
    l = config.logging_dict()
    assert 'DEBUG' == l['loggers']['ldap2pg']['level']

    config['verbose'] = False
    l = config.logging_dict()
    assert 'INFO' == l['loggers']['ldap2pg']['level']


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
        environ=dict(MY_OPTION='envval'),
    )
    assert 'fileval' == v

    m = Mapping('my:option')
    assert 'MY_OPTION' in m.env
    assert 'MYOPTION' in m.env

    # Prefer env over file
    v = m.process(
        default='defval',
        file_config=dict(my=dict(option='fileval')),
        environ=dict(MY_OPTION='envval'),
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


def test_processor():
    from ldap2pg.config import Mapping

    m = Mapping('dry', processor=bool)
    v = m.process(default=True, file_config=dict(dry=0), environ=dict())

    assert v is False


def test_process_acldict():
    from ldap2pg.config import acldict

    with pytest.raises(ValueError):
        acldict([])

    acl_dict = acldict(dict(ro=dict(inspect='SQL', grant='SQL', revoke='SQL')))

    assert 'ro' in acl_dict


def test_process_grant():
    from ldap2pg.config import grantrule

    with pytest.raises(ValueError):
        grantrule([])

    with pytest.raises(ValueError):
        grantrule(dict(missing_acl=True))

    with pytest.raises(ValueError):
        grantrule(dict(acl='toto', spurious_attribute=True))

    with pytest.raises(ValueError):
        grantrule(dict(acl='missing role*'))

    grantrule(dict(
        acl='ro',
        database='postgres',
        schema='public',
        role_attribute='cn',
    ))


def test_ismapping():
    from ldap2pg.config import ismapping

    assert ismapping(dict(ldap=dict()))
    assert ismapping(dict(roles=[]))
    assert ismapping(dict(role=dict()))
    assert not ismapping([])
    assert not ismapping(dict(__common__=[]))


def test_process_syncmap():
    from ldap2pg.config import syncmap

    fixtures = [
        # Canonical case.
        dict(
            __common__=dict(
                __common__=[
                    dict(role=dict(name='alice')),
                ]
            ),
        ),
        # Squeeze list.
        dict(
            __common__=dict(
                __common__=dict(role=dict(name='alice')),
            ),
        ),
        # Squeeze also schema.
        dict(__common__=dict(role=dict(name='alice'))),
        # Squeeze also database.
        dict(role=dict(name='alice')),
        # Direct list (this is 1.0 format).
        [dict(role=dict(name='alice'))],
    ]

    for raw in fixtures:
        v = syncmap(raw)

        assert isinstance(v, dict)
        assert '__common__' in v
        assert '__common__' in v['__common__']
        maplist = v['__common__']['__common__']
        assert 1 == len(maplist)
        assert 'roles' in maplist[0]

    # Missing rules
    raw = dict(ldap=dict(base='dc=unit', attribute='cn'))
    with pytest.raises(ValueError):
        syncmap(raw)

    bad_fixtures = [
        'string_value',
        [None],
    ]
    for raw in bad_fixtures:
        with pytest.raises(ValueError):
            syncmap(raw)


def test_process_mapping_grant():
    from ldap2pg.config import mapping

    mapping(dict(grant=dict(acl='ro', role='alice')))


def test_process_ldapquery():
    from ldap2pg.config import ldapquery

    raw = dict(base='dc=unit', attribute='cn')

    v = ldapquery(raw)

    assert 'attributes' in v
    assert 'attribute' not in v
    assert 'filter' in v


def test_process_rolerule():
    from ldap2pg.config import rolerule

    rule = rolerule('aline')
    assert 'aline' == rule['names'][0]

    rule = rolerule(dict(name='rolname', parent='parent'))
    assert ['rolname'] == rule['names']
    assert ['parent'] == rule['parents']

    rule = rolerule(dict(options='LOGIN SUPERUSER'))
    assert rule['options']['LOGIN'] is True
    assert rule['options']['SUPERUSER'] is True
    assert 'names' not in rule

    rule = rolerule(dict(options=['LOGIN', 'SUPERUSER']))
    assert rule['options']['LOGIN'] is True
    assert rule['options']['SUPERUSER'] is True

    rule = rolerule(dict(options=['NOLOGIN', 'SUPERUSER']))
    assert rule['options']['LOGIN'] is False
    assert rule['options']['SUPERUSER'] is True

    with pytest.raises(ValueError) as ei:
        rolerule(dict(options='OLOLOL'))
    assert 'OLOLOL' in str(ei.value)


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
        config.find_filename(environ=dict(LDAP2PG_CONFIG='my.yml'))

    # Read from args
    stat.reset_mock()
    stat.side_effect = [
        mocker.Mock(st_mode=0o600),
        AssertionError("Not reached."),
    ]
    filename, mode = config.find_filename(
        environ=dict(LDAP2PG_CONFIG='env.yml'),
        args=MockArgs(config='argv.yml'),
    )

    assert filename.endswith('argv.yml')


def test_find_filename_stdin():
    from ldap2pg.config import Configuration

    config = Configuration()

    filename, mode = config.find_filename(
        environ=dict(LDAP2PG_CONFIG='-'),
    )

    assert '-' == filename
    assert 0o400 == mode


def test_merge_and_mappings():
    from ldap2pg.config import Configuration

    # Noop
    config = Configuration()
    with pytest.raises(ValueError):
        config.merge(file_config={}, environ={})

    # Minimal configuration
    minimal_config = dict(
        ldap=dict(host='confighost'),
        sync_map=dict(ldap=dict(), role=dict()),
    )
    config.merge(
        file_config=minimal_config,
        environ=dict(),
    )
    config.merge(
        file_config=minimal_config,
        environ=dict(LDAP_PASSWORD='envpass', PGDSN='envdsn'),
    )
    assert 'confighost' == config['ldap']['host']
    assert 'envpass' == config['ldap']['password']
    assert 'envdsn' == config['postgres']['dsn']


def test_security():
    from ldap2pg.config import Configuration

    config = Configuration()

    minimal_config = dict(
        ldap=dict(host='confighost'),
        sync_map=dict(ldap=dict(), role=dict()),
    )

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
    payload = config.read(fo, mode=0o0)
    assert 'sync_map' in payload

    fo = StringIO("entry: value")
    payload = config.read(fo, mode=0o644)
    assert 'entry' in payload
    assert payload['world_readable'] is True

    # Accept empty file (e.g. /dev/null)
    fo = StringIO("")
    payload = config.read(fo, mode=0o600)
    assert payload['world_readable'] is False

    with pytest.raises(ConfigurationError):
        fo = StringIO("bad_value")
        payload = config.read(fo, mode=0o600)


def test_load_badfiles(mocker):
    environ = dict()
    mocker.patch('ldap2pg.config.os.environ', environ)
    ff = mocker.patch('ldap2pg.config.Configuration.find_filename')
    o = mocker.patch('ldap2pg.config.open', create=True)

    from ldap2pg.config import (
        Configuration,
        ConfigurationError,
        NoConfigurationError,
        UserError,
    )

    config = Configuration()

    ff.side_effect = NoConfigurationError()
    # Missing sync_map
    with pytest.raises(ConfigurationError):
        config.load(argv=[])

    ff.side_effect = None
    # Find `filename.yml`
    ff.return_value = ['filename.yml', 0o0]

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

    maplist = config['sync_map']['__common__']['__common__']
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
    environ.update(dict(LDAP_BIND='envbind'))

    config.load(argv=['--verbose'])

    assert 'envbind' == config['ldap']['bind']
    maplist = config['sync_map']['__common__']['__common__']
    assert 1 == len(maplist)
    assert config['verbose'] is True
