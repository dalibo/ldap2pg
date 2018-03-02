from __future__ import print_function
from __future__ import unicode_literals

from argparse import ArgumentParser, SUPPRESS as SUPPRESS_ARG
from textwrap import dedent
from argparse import _VersionAction
from pkg_resources import get_distribution
from codecs import open
import errno
import logging
try:
    from logging.config import dictConfig
except ImportError:  # pragma: nocover
    from logutils.dictconfig import dictConfig

import os.path
from os import stat
import re
import sys

import psycopg2
import yaml

from . import __version__
from .acl import Acl
from .acl import process_definitions as process_acls
from .utils import (
    deepget,
    deepset,
    UserError,
    string_types,
)
from . import validators as V
from .defaults import make_well_known_acls


logger = logging.getLogger(__name__)


class MultilineFormatter(logging.Formatter):
    def format(self, record):
        s = logging.Formatter.format(self, record)
        if '\n' not in s:
            return s

        lines = s.splitlines()
        d = record.__dict__.copy()
        for i, line in enumerate(lines[1:]):
            record.message = line
            lines[1+i] = self._fmt % record.__dict__
        record.__dict__ = d

        return '\n'.join(lines)


class ColoredStreamHandler(logging.StreamHandler):

    _color_map = {
        logging.DEBUG: '37',
        logging.INFO: '1;39',
        logging.WARN: '96',
        logging.ERROR: '91',
        logging.CRITICAL: '1;91',
    }

    def format(self, record):
        lines = logging.StreamHandler.format(self, record)
        color = self._color_map.get(record.levelno, '39')
        lines = ''.join([
            '\033[0;%sm%s\033[0m' % (color, line)
            for line in lines.splitlines(True)
        ])
        return lines


class VersionAction(_VersionAction):
    def __call__(self, parser, *a):
        try:
            pyldap = get_distribution('pyldap')
        except Exception:  # pragma: nocover_py3
            pyldap = get_distribution('python-ldap')

        version = (
            "%(package)s %(version)s\n"
            "psycopg2 %(psycopg2version)s\n"
            "%(pyldap)s %(ldapversion)s\n"
            "Python %(pyversion)s\n"
        ) % dict(
            package=__package__,
            version=__version__,
            psycopg2version=psycopg2.__version__,
            pyversion=sys.version,
            pyldap=pyldap.project_name,
            ldapversion=pyldap.version,

        )
        print(version.strip())
        parser.exit()


def define_arguments(parser):
    parser.add_argument(
        '-c', '--config',
        action='store', dest='config', metavar='PATH',
        help=(
            'path to YAML configuration file (env: LDAP2PG_CONFIG). '
            'Use - for stdin.'
        )
    )
    parser.add_argument(
        '-C', '--check',
        action='store_true', dest='check',
        help='check mode: exits with 1 on changes in cluster',
    )
    parser.add_argument(
        '-n', '--dry',
        action='store_true', dest='dry',
        help="don't touch Postgres, just print what to do (env: DRY=1)"
    )
    parser.add_argument(
        '-N', '--real',
        action='store_false', dest='dry',
        help="real mode, apply changes to Postgres (env: DRY='')"
    )
    parser.add_argument(
        '-q', '--quiet',
        action='store_false', dest='verbose',
        help="hide debugging messages",
    )
    parser.add_argument(
        '-v', '--verbose',
        action='store_true', dest='verbose',
        help="add debug messages including SQL and LDAP queries (env: VERBOSE)"
    )
    parser.add_argument(
        '--color',
        action='store_true', dest='color',
        help="force color output (env: COLOR=1)"
    )
    parser.add_argument(
        '--no-color',
        action='store_false', dest='color',
        help="force plain text output (env: COLOR='')"
    )
    parser.add_argument(
        '-?', '--help',
        action='help',
        help='show this help message and exit')

    parser.add_argument(
        '-V', '--version',
        action=VersionAction,
        help='show version and exit',
    )


def merge_acl_options(acls, acl_dict, acl_groups):
    final = dict()
    final.update(acl_dict)
    final.update(acl_groups)
    final.update(acls)
    return V.acls(final)


def list_unused_acl(acls, aliases):
    used = set()
    for name, aliases in aliases.items():
        if name[0] not in ('_', '.'):
            used.add(name)
            used.update(aliases)
    unused = set(acls.keys()) - used
    return sorted(unused)


def postprocess_acl_options(self, defaults=None):
    # Compat with user defined acl_dict and acl_groups, merge in the same
    # namespace.
    acls = defaults or {}
    acls.update(self.pop('acls', {}))
    acls = merge_acl_options(
        acls,
        self.get('acl_dict', {}),
        self.pop('acl_groups', {}),
    )

    acls, _, self['acl_aliases'] = process_acls(acls)

    # Clean unused ACL starting with _ or .
    for k in list_unused_acl(acls, self['acl_aliases']):
        logger.debug("Drop unused hidden ACL %s", k)
        del acls[k]

    self['acl_dict'] = dict([
        (k, Acl.factory(k, **v)) for k, v in acls.items()
    ])


class Mapping(object):
    """Fetch value from either file or env var."""

    _auto_env = object()

    def __init__(self, path, env=_auto_env, secret=False, processor=V.raw):
        self.path = path
        self.arg = path.replace(':', '_')

        env = env or []
        if env == self._auto_env:
            env = [self.arg.upper(), self.path.upper().replace(':', '')]
        self.env = env
        if isinstance(self.env, string_types):
            self.env = [self.env]

        self.processor = processor
        if isinstance(secret, string_types):
            secret = re.compile(secret)
        self.secret = secret

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.path)

    def process_env(self, environ):
        # Get value from env var
        for env in self.env:
            try:
                value = environ[env]
                if hasattr(value, 'decode'):
                    value = value.decode('utf-8')
                logger.debug("Read %s from %s.", self.path, env)
                break
            except KeyError:
                continue
        else:
            raise KeyError()

        return value

    def process_file(self, file_config):
        # Get value from parsed YAML file.
        unsecured_file = file_config.get('world_readable', True)

        value = deepget(file_config, self.path)

        # Check whether this value is secret.
        if hasattr(self.secret, 'search'):
            secret = self.secret.search(value)
        else:
            secret = self.secret

        if secret and unsecured_file:
            msg = "Refuse to load %s from world readable file." % (self.path)
            raise ValueError(msg)

        logger.debug("Read %s from YAML.", self.path)
        return value

    def process_arg(self, args):
        # Get value from argparse result.
        value = getattr(args, self.arg)
        logger.debug("Read %s from argv.", self.path)
        return value

    def process(self, default, file_config={}, environ={}, args=object()):
        # This is the sources of configuration, ordered by priority desc. If a
        # process_* function raises KeyError or AttributeError, it is ignored.
        sources = [
            (self.process_arg, args),
            (self.process_env, environ),
            (self.process_file, file_config),
        ]

        for source in sources:
            callable_, args = source[0], source[1:]
            try:
                value = callable_(*args)
                break
            except (AttributeError, KeyError):
                continue
        else:
            value = default

        return self.processor(value)


class ConfigurationError(UserError):
    def __init__(self, message):
        super(ConfigurationError, self).__init__(
            message, exit_code=os.EX_CONFIG,
        )


class NoConfigurationError(Exception):
    pass


def construct_yaml_str(self, node):
    # See https://stackoverflow.com/a/2967461/2613806
    return self.construct_scalar(node)


yaml.Loader.add_constructor(u'tag:yaml.org,2002:str', construct_yaml_str)


class Configuration(dict):
    DEFAULTS = {
        'check': False,
        'dry': True,
        'verbose': False,
        'color': False,
        'ldap': {
            'uri': '',
            'host': '',
            'port': 389,
            'binddn': '',
            'user': None,
            'password': '',
        },
        'postgres': {
            'dsn': '',
            'blacklist': ['pg_*', 'postgres'],
            'databases_query': dedent("""\
            SELECT datname FROM pg_catalog.pg_database
            WHERE datallowconn IS TRUE ORDER BY 1;
            """),
            # SQL Query to inspect roles in cluster. See
            # https://www.postgresql.org/docs/current/static/view-pg-roles.html
            # and
            # https://www.postgresql.org/docs/current/static/catalog-pg-auth-members.html
            'roles_query': dedent("""\
            SELECT
              role.rolname, array_agg(members.rolname) AS members, {options}
            FROM
              pg_catalog.pg_roles AS role
            LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
            LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
            GROUP BY role.rolname, {options}
            ORDER BY 1;
            """),
            'owners_query': dedent("""\
            SELECT role.rolname
            FROM pg_catalog.pg_roles AS role
            WHERE role.rolsuper IS TRUE
            ORDER BY 1;
            """),
            'managed_roles_query': None,
            'schemas_query': dedent("""\
            SELECT nspname FROM pg_catalog.pg_namespace
            ORDER BY 1;
            """),
        },
        'acls': {},
        'acl_dict': {},
        'acl_groups': {},
        'sync_map': {},
    }

    MAPPINGS = [
        Mapping('color'),
        Mapping('check'),
        Mapping('dry'),
        Mapping('verbose', env=['VERBOSE', 'DEBUG']),
        Mapping('ldap:uri'),
        Mapping('ldap:host'),
        Mapping('ldap:port'),
        Mapping('ldap:binddn', env=['LDAPBINDDN', 'LDAP_BIND']),
        Mapping('ldap:user'),
        Mapping('ldap:password', secret=True),
        Mapping(
            'postgres:dsn', env='PGDSN',
            secret=r'(?:password=|:[^/][^/].*@)',
        ),
        Mapping('postgres:blacklist', env=None),
        Mapping('postgres:databases_query', env=None),
        Mapping('postgres:owners_query', env=None),
        Mapping('postgres:roles_query', env=None),
        Mapping('postgres:managed_roles_query', env=None),
        Mapping('postgres:schemas_query', env=None),
        Mapping('acls', env=None, processor=V.acls),
        Mapping('acl_dict', processor=V.acldict),
        Mapping('acl_groups', env=None),
        Mapping('sync_map', env=None, processor=V.syncmap)
    ]

    def __init__(self):
        super(Configuration, self).__init__(self.DEFAULTS)

    _file_candidates = [
        './ldap2pg.yml',
        '~/.config/lda2pg.yml',
        '/etc/ldap2pg.yml',
    ]

    def find_filename(self, environ=os.environ, args=None):
        custom = getattr(
            args, 'config',
            environ.get('LDAP2PG_CONFIG', ''),
        )

        if hasattr(custom, 'decode'):
            custom = custom.decode('utf-8')

        if '-' == custom:
            return custom, 0o400
        elif custom:
            candidates = [custom]
        else:
            candidates = self._file_candidates

        for candidate in candidates:
            candidate = os.path.expanduser(candidate)
            try:
                logger.debug("Trying %s.", candidate)
                stat_ = stat(candidate)
                return os.path.realpath(candidate), stat_.st_mode
            except OSError as e:
                if e.errno == errno.EACCES:
                    logger.warn("Can't read %s: permission denied.", candidate)

        if custom:
            message = "Can't access configuration file %s." % (custom,)
            raise UserError(message, exit_code=os.EX_NOINPUT)
        else:
            raise NoConfigurationError("No configuration file found")

    EPILOG = dedent("""\

    ldap2pg requires a configuration file to describe LDAP queries and role
    mappings. See https://ldap2pg.readthedocs.io/en/latest/ for further
    details.

    By default, ldap2pg runs in dry mode.
    """)

    def has_ldap_query(self):
        return [m['ldap'] for m in self['sync_map'] if 'ldap' in m]

    def load(self, argv=None):
        # argv processing.
        logger.debug("Processing CLI arguments.")
        parser = ArgumentParser(
            add_help=False,
            # Only store value from argv. Defaults are managed by
            # Configuration.
            argument_default=SUPPRESS_ARG,
            description="PostgreSQL roles and ACL management.",
            epilog=self.EPILOG,
        )
        define_arguments(parser)
        args = parser.parse_args(sys.argv[1:] if argv is None else argv)

        if hasattr(args, 'verbose') or hasattr(args, 'color'):
            # Switch to verbose before loading file.
            self['verbose'] = getattr(args, 'verbose', self['verbose'])
            self['color'] = getattr(args, 'color', self['color'])
            dictConfig(self.logging_dict())

        logger.info("Starting ldap2pg %s.", __version__)

        # File loading.
        try:
            filename, mode = self.find_filename(os.environ, args)
        except NoConfigurationError:
            logger.debug("No configuration file found.")
            file_config = {}
        else:
            if filename == '-':
                logger.info("Reading configuration from stdin.")
                file_config = self.read(sys.stdin, 'stdin', mode)
            else:
                logger.info("Using %s.", filename)
                try:
                    with open(filename, encoding='utf-8') as fo:
                        file_config = self.read(fo, filename, mode)
                except OSError as e:
                    msg = "Failed to read configuration: %s" % (e,)
                    raise UserError(msg)

        # Now close stdin. To make SASL non-interactive.
        if not self.get('debug'):
            sys.stdin.close()

        # Now merge all config sources.
        acl_defaults = make_well_known_acls()
        try:
            self.merge(file_config=file_config, environ=os.environ, args=args)
            postprocess_acl_options(self, acl_defaults)
        except ValueError as e:
            raise ConfigurationError("Failed to load configuration: %s" % (e,))

        logger.debug("Configuration loaded.")

        if not self['sync_map']:
            logger.warn("Empty synchronization map!")

    def merge(self, file_config, environ=os.environ, args=object()):
        for mapping in self.MAPPINGS:
            value = mapping.process(
                default=deepget(self, mapping.path),
                file_config=file_config,
                environ=environ,
                args=args,
            )
            deepset(self, mapping.path, value)

    def read(self, fo, name, mode):
        try:
            payload = yaml.load(fo) or {}
        except yaml.error.YAMLError as e:
            msg = "YAML error with %s: %s" % (name, e)
            raise ConfigurationError(msg)

        if isinstance(payload, list):
            payload = dict(sync_map=payload)
        if not isinstance(payload, dict):
            raise ConfigurationError("Configuration file must be a mapping.")
        payload['world_readable'] = bool(mode & 0o077)
        return payload

    def logging_dict(self):
        formatter = 'verbose' if self['verbose'] else 'info'

        return {
            'version': 1,
            'formatters': {
                'info': {
                    '()': __name__ + '.MultilineFormatter',
                    'format': '%(message)s',
                },
                'verbose': {
                    '()': __name__ + '.MultilineFormatter',
                    'format': '[%(name)-20s %(levelname)5.5s] %(message)s',
                },
            },
            'handlers': {
                'raw': {
                    '()': 'logging.StreamHandler',
                    'formatter': formatter,
                },
                'colored': {
                    '()': __name__ + '.ColoredStreamHandler',
                    'formatter': formatter,
                },
            },
            'root': {
                'level': 'WARNING',
                'handlers': ['colored' if self['color'] else 'raw'],
            },
            'loggers': {
                __package__: {
                    'level': 'DEBUG' if self['verbose'] else 'INFO',
                },
            },
        }
