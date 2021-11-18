from __future__ import unicode_literals

from textwrap import dedent

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
    wanted = dedent("""\
    prefix: Uno
    prefix: Dos
    prefix: Tres
    """).strip()

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
    from ldap2pg.config import Configuration, UserError

    config = Configuration()

    config['verbosity'] = 'DEBUG'
    dict_ = config.logging_dict()
    assert 'DEBUG' == dict_['loggers']['ldap2pg']['level']

    with pytest.raises(UserError):
        config.bootstrap(environ=dict(VERBOSITY='TOTO'))


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

    from ldap2pg.config import Configuration, ConfigurationError

    config = Configuration()

    def mk_oserror(errno=None):
        e = OSError()
        e.errno = errno
        return e

    # Search default path
    stat.side_effect = [
        mk_oserror(),
        mk_oserror(),
        mk_oserror(13),
        mk_oserror(13),
        mocker.Mock(st_mode=0o600),
    ]
    filename, mode = config.find_filename(environ=dict())
    assert config._file_candidates[4] == filename
    assert 0o600 == mode

    # No files at all
    stat.side_effect = OSError()
    with pytest.raises(ConfigurationError):
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

    minimal_config = dict(verbose=True, sync_map=[])
    config = Configuration()
    config.merge(
        file_config=minimal_config,
        environ=dict(),
    )
    config = Configuration()
    config.merge(
        file_config=minimal_config,
        environ=dict(PGDSN=b'envdsn'),
    )
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

    fo = StringIO("sync_map: []")
    payload = config.read(fo, 'memory', mode=0o644)
    assert 'sync_map' in payload
    assert payload['world_readable'] is True

    # Refuse empty file (e.g. /dev/null)
    with pytest.raises(ConfigurationError):
        fo = StringIO("")
        config.read(fo, 'memory', mode=0o600)

    with pytest.raises(ConfigurationError):
        fo = StringIO("bad_value")
        payload = config.read(fo, 'memory', mode=0o600)

    with pytest.raises(ConfigurationError):
        fo = StringIO("bad: { yaml ] *&")
        payload = config.read(fo, 'memory', mode=0o600)

    # No sync_map.
    with pytest.raises(ConfigurationError):
        fo = StringIO("postgres: {}")
        payload = config.read(fo, 'memory', mode=0o600)


def test_load_badfiles(mocker):
    environ = dict()
    mocker.patch('ldap2pg.config.os.environ', environ)
    ff = mocker.patch('ldap2pg.config.Configuration.find_filename')
    merge = mocker.patch('ldap2pg.config.Configuration.merge')
    read = mocker.patch('ldap2pg.config.Configuration.read')

    from ldap2pg.config import (
        Configuration,
        ConfigurationError,
        UserError,
    )

    config = Configuration()

    # Invalid file
    ff.return_value = ['filename.yml', 0o0]
    merge.side_effect = ValueError()
    o = mocker.patch('ldap2pg.config.open', mocker.mock_open(), create=True)
    read.return_value = {}
    with pytest.raises(ConfigurationError):
        config.load(argv=[])

    # Not readable.
    o.side_effect = OSError("failed to open")
    with pytest.raises(UserError):
        config.load(argv=["--color"])


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

    maplist = config['sync_map']
    assert 1 == len(maplist)


def test_load_file(mocker):
    environ = dict()
    mod = 'ldap2pg.config.'
    mocker.patch(mod + 'os.environ', environ)
    ff = mocker.patch(mod + 'Configuration.find_filename')
    mocker.patch(mod + 'open', create=True)
    read = mocker.patch(mod + 'Configuration.read')
    warn = mocker.patch(mod + 'logger.warning')

    from ldap2pg.config import Configuration

    config = Configuration()

    ff.return_value = ['filename.yml', 0o0]
    read.return_value = dict(
        sync_map=[dict(role='alice')],
        # To trigger a warning.
        unknown_key=True,
        # Should not trigger a warning.
        privileges=dict(ro=['__connect__']),
    )
    # send one env var
    environ.update(dict(PGDSN=b'envdsn'))

    config.load(argv=['--verbose'])

    assert 'envdsn' == config['postgres']['dsn']
    maplist = config['sync_map']
    assert 1 == len(maplist)
    assert 'DEBUG' == config['verbosity']
    # logger.warning is called once for unknown_key, not for privileges.
    assert 1 == warn.call_count


def test_show_versions(mocker):
    from ldap2pg.config import Configuration

    config = Configuration()
    with pytest.raises(SystemExit):
        config.load(argv=['--version'])


def test_has_ldapsearch():
    from ldap2pg.config import Configuration

    config = Configuration()

    config['sync_map'] = [dict(roles=dict())]
    assert not config.has_ldapsearch()

    config['sync_map'] = [dict(ldapsearch=dict())]
    assert config.has_ldapsearch()


def test_privilege_options():
    from ldap2pg.config import postprocess_privilege_options

    config_v33 = dict(
        acl_dict=dict(
            select=dict(type='nspacl', inspect='INSPECT'),
        ),
        acl_groups=dict(ro=['select'])
    )

    postprocess_privilege_options(config_v33)

    config_v34 = dict(acls=dict(
        select=dict(type='nspacl', inspect='INSPECT'),
        ro=['select'],
    ))

    postprocess_privilege_options(config_v34)

    assert config_v33 == config_v34

    config_v49 = dict(privileges=dict(
        select=dict(type='nspacl', inspect='INSPECT'),
        ro=['select'],
    ))

    postprocess_privilege_options(config_v49)

    assert config_v49 == config_v34


def test_yaml_gotchas():
    from ldap2pg.config import ConfigurationError, check_yaml_gotchas

    config = dict(postgres=dict(dsn='', none_query=None))
    check_yaml_gotchas(config)

    # When postgres: entries are not indented.
    config = dict(postgres=None)
    with pytest.raises(ConfigurationError):
        check_yaml_gotchas(config)

    # When herestring for bad_query is not indented
    config = dict(postgres=dict(bad_query=''))
    with pytest.raises(ConfigurationError):
        check_yaml_gotchas(config)


def test_extract_static_rules_roles():
    from ldap2pg.config import extract_static_rules
    from ldap2pg.validators import rolerule

    config = dict(sync_map=[
        dict(
            ldap=dict(filter="(filter)"),
            roles=[
                rolerule(dict(name="static-orphan")),
                rolerule(dict(name="static", parent=["static"])),
                rolerule(dict(name="{dynamic}")),
                rolerule(dict(names=["mixed", "{dynamic}"])),
                rolerule(dict(name="dynmember", members=["{dynamic}"])),
                rolerule(dict(name="dynparent", parent=["{dynamic}"])),
                rolerule(dict(name="dyncomment", comment="{dynamic}")),
            ],
        ),
    ])

    extract_static_rules(config)

    wanted = dict(sync_map=[
        dict(roles=[
                rolerule(dict(name="static-orphan")),
        ]),
        dict(roles=[
                rolerule(dict(name="static", parent=["static"])),
        ]),
        dict(roles=[
                rolerule(dict(name="mixed")),
        ]),
        dict(
            ldap=dict(filter="(filter)"),
            roles=[
                rolerule(dict(name="{dynamic}")),
                rolerule(dict(names=["{dynamic}"])),
                rolerule(dict(name="dynmember", members=["{dynamic}"])),
                rolerule(dict(name="dynparent", parent=["{dynamic}"])),
                rolerule(dict(name="dyncomment", comment="{dynamic}")),
            ],
        ),
    ])

    assert wanted == config


def test_extract_static_rules_grants():
    from ldap2pg.config import extract_static_rules
    from ldap2pg.validators import grantrule

    kw = dict(privilege='ro')
    config = dict(sync_map=[
        dict(
            ldap=dict(filter="(filter)"),
            grants=[
                grantrule(dict(role="static", database=["static"], **kw)),
                grantrule(dict(role="{dynamic}", **kw)),
                grantrule(dict(roles=["mixed", "{dynamic}"], **kw)),
                grantrule(dict(role="dyndatabase", database="{dyn}", **kw)),
                grantrule(dict(role="dynschema", schema="{dynamic}", **kw)),
                grantrule(dict(role="dynpriv", privilege="{dynamic}")),
            ],
        ),
    ])

    extract_static_rules(config)

    wanted = dict(sync_map=[
        dict(grants=[
                grantrule(dict(role="static", database=["static"], **kw)),
        ]),
        dict(grants=[
                grantrule(dict(role="mixed", **kw)),
        ]),
        dict(
            ldap=dict(filter="(filter)"),
            grants=[
                grantrule(dict(role="{dynamic}", **kw)),
                grantrule(dict(roles=["{dynamic}"], **kw)),
                grantrule(dict(role="dyndatabase", database="{dyn}", **kw)),
                grantrule(dict(role="dynschema", schema="{dynamic}", **kw)),
                grantrule(dict(role="dynpriv", privilege="{dynamic}")),
            ],
        ),
    ])

    assert wanted == config


def test_format_pq_version():
    from ldap2pg.config import VersionAction

    assert '14.1' == VersionAction.format_pq_version(140001)
    assert '13.5' == VersionAction.format_pq_version(130005)
    assert '12.9' == VersionAction.format_pq_version(120009)
    assert '11.14' == VersionAction.format_pq_version(110014)
    assert '10.19' == VersionAction.format_pq_version(100019)
    assert '9.6.24' == VersionAction.format_pq_version(90624)
