from __future__ import absolute_import
from __future__ import print_function
from __future__ import unicode_literals

from argparse import ArgumentParser, SUPPRESS as SUPPRESS_ARG
from textwrap import dedent
from argparse import _VersionAction
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

import ldap
import psycopg2
import yaml

from . import __version__
from .privilege import Privilege
from .privilege import process_definitions as process_privileges
from .utils import (
    deepget,
    deepset,
    iter_deep_keys,
    UserError,
    string_types,
)
from . import validators as V
from .defaults import make_well_known_privileges
from .defaults import shared_queries as default_shared_queries


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
        logging.CHANGE: '1;39',
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
        version = (
            "%(package)s %(version)s\n"
            "psycopg2 %(psycopg2version)s\n"
            "python-ldap %(ldapversion)s\n"
            "Python %(pyversion)s\n"
        ) % dict(
            package=__package__,
            version=__version__,
            psycopg2version=psycopg2.__version__,
            pyversion=sys.version,
            ldapversion=ldap.__version__,

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
        action='append_const', dest='verbosity', const=-1,
        default=[V.VERBOSITIES.index(Configuration.DEFAULTS['verbosity'])],
        help="decrease log verbosity (env: VERBOSITY)",
    )
    parser.add_argument(
        '-v', '--verbose',
        action='append_const', dest='verbosity', const=+1,
        help="increase log verbosity (env: VERBOSITY)"
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


def merge_privilege_options(privileges, acl_dict, acl_groups):
    final = dict()
    final.update(acl_dict)
    final.update(acl_groups)
    final.update(privileges)
    return V.privileges(final)


def list_unused_privilege(privileges, aliases):
    used = set()
    for name, aliases in aliases.items():
        if name[0] not in ('_', '.'):
            used.add(name)
            used.update(aliases)
    unused = set(privileges.keys()) - used
    return sorted(unused)


def postprocess_privilege_options(self, defaults=None):
    # Inject default shared_queries.
    self.setdefault('postgres', {})
    conf_shared_queries = self['postgres'].get('shared_queries', {})
    self['postgres']['shared_queries'] = dict(
        default_shared_queries, **conf_shared_queries)

    # Compat with user defined acl_dict and acl_groups, merge in the same
    # namespace.
    privileges = defaults or {}
    privileges.update(self.pop('acls', {}))
    privileges.update(self.pop('privileges', {}))
    privileges = merge_privilege_options(
        privileges,
        self.pop('acl_dict', {}),
        self.pop('acl_groups', {}),
    )

    privileges, _, self['privilege_aliases'] = process_privileges(privileges)

    # Clean unused privilege starting with _ or .
    for k in list_unused_privilege(privileges, self['privilege_aliases']):
        logger.debug("Drop unused hidden privilege %s", k)
        del privileges[k]

    self['privileges'] = dict([
        (k, Privilege.factory(k, **v)) for k, v in privileges.items()
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


def construct_yaml_str(self, node):
    # See https://stackoverflow.com/a/2967461/2613806
    return self.construct_scalar(node)


yaml.SafeLoader.add_constructor(u'tag:yaml.org,2002:str', construct_yaml_str)


def check_yaml_gotchas(file_config):
    dict_keys = ('ldap', 'postgres')
    for key in dict_keys:
        if key not in file_config:
            continue

        if not hasattr(file_config[key], 'items'):
            msg = "Error ni YAML: %s: is not a dict." % key
            raise ConfigurationError(msg)

    for k, v in file_config.get('postgres', {}).items():
        if not k.endswith('_query'):
            continue
        if v is None:
            continue
        if not v:
            msg = "Error in YAML: postgres:%s: is empty." % k
            raise ConfigurationError(msg)


class Configuration(dict):
    DEFAULTS = {
        'check': False,
        'dry': True,
        'verbose': None,
        'verbosity': 'INFO',
        'color': False,
        'ldap': {
            'uri': None,
            'host': None,
            'port': None,
            'binddn': None,
            'user': None,
            'password': None,
            'referrals': None,
        },
        'postgres': {
            'dsn': '',
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
              role.rolname, array_agg(members.rolname) AS members,
              {options},
              pg_catalog.shobj_description(role.oid, 'pg_authid') as comment
            FROM
              pg_catalog.pg_roles AS role
            LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
            LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
            GROUP BY role.rolname, {options}, comment
            ORDER BY 1;
            """),
            'owners_query': dedent("""\
            SELECT role.rolname
            FROM pg_catalog.pg_roles AS role
            WHERE role.rolsuper IS TRUE
            ORDER BY 1;
            """),
            'managed_roles_query': None,
            'roles_blacklist_query': [
                'pg_*', 'postgres', 'rds_*', 'rds*admin',
            ],
            'schemas_query': dedent("""\
            SELECT nspname FROM pg_catalog.pg_namespace
            ORDER BY 1;
            """),
            'shared_queries': {},
        },
        'privileges': {},
        'acls': {},
        'acl_dict': {},
        'acl_groups': {},
        'sync_map': {},
    }

    MAPPINGS = [
        Mapping('color'),
        Mapping('check'),
        Mapping('dry'),
        Mapping('verbose', env=[]),
        Mapping('verbosity', processor=V.verbosity),
        Mapping('ldap:uri'),
        Mapping('ldap:host'),
        Mapping('ldap:port'),
        Mapping('ldap:binddn', env=['LDAPBINDDN', 'LDAP_BIND']),
        Mapping('ldap:user'),
        Mapping('ldap:password', secret=True),
        Mapping('ldap:referrals'),
        Mapping(
            'postgres:dsn', env='PGDSN',
            secret=r'(?:password=|:[^/][^/].*@)',
        ),
        Mapping('postgres:databases_query', env=None),
        Mapping('postgres:owners_query', env=None),
        Mapping('postgres:roles_query', env=None),
        Mapping('postgres:managed_roles_query', env=None),
        Mapping('postgres:roles_blacklist_query', env=None),
        Mapping('postgres:schemas_query', env=None),
        Mapping(
            'postgres:shared_queries', processor=V.shared_queries, env=None),
        Mapping('privileges', env=None, processor=V.privileges),
        Mapping('acls', env=None, processor=V.privileges),
        Mapping('acl_dict', processor=V.privileges),
        Mapping('acl_groups', env=None),
        Mapping('sync_map', env=None, processor=V.syncmap)
    ]

    def __init__(self):
        super(Configuration, self).__init__(self.DEFAULTS)

    _file_candidates = [
        './ldap2pg.yml',
        '~/.config/ldap2pg.yml',
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
                    logger.warning(
                        "Can't read %s: permission denied.", candidate)

        if custom:
            message = "Can't access configuration file %s." % (custom,)
            raise UserError(message, exit_code=os.EX_NOINPUT)
        else:
            raise ConfigurationError("No configuration file found")

    EPILOG = dedent("""\

    ldap2pg requires a configuration file to describe LDAP queries and role
    mappings. See https://ldap2pg.readthedocs.io/en/latest/ for further
    details.

    By default, ldap2pg runs in dry mode.
    """)

    def has_ldap_query(self):
        return [m['ldap'] for m in self['sync_map'] if 'ldap' in m]

    def bootstrap(self, environ=os.environ):
        debug = environ.get('DEBUG', '').lower() in ('1', 'y')
        verbose = debug or environ.get('VERBOSE', '').lower() in ('1', 'y')
        verbosity = environ.get('VERBOSITY', 'DEBUG' if verbose else 'INFO')

        self['debug'] = debug
        try:
            self['verbosity'] = V.verbosity(verbosity)
        except ValueError as e:
            raise UserError('Failed to boostrap: %s.' % (e,))
        self['color'] = sys.stderr.isatty()

        dictConfig(self.logging_dict())
        return debug

    def load(self, argv=None):
        # argv processing.
        logger.debug("Processing CLI arguments.")
        parser = ArgumentParser(
            add_help=False,
            # Only store value from argv. Defaults are managed by
            # Configuration.
            argument_default=SUPPRESS_ARG,
            description="PostgreSQL roles and privileges management.",
            epilog=self.EPILOG,
        )
        define_arguments(parser)
        args = parser.parse_args(sys.argv[1:] if argv is None else argv)

        # Setup logging before parsing options. Reset verbosity with env var,
        # and compute verbosity from cumulated args.
        args.verbosity[0] = V.VERBOSITIES.index(self['verbosity'])
        self['verbosity'] = V.verbosity(args.verbosity)
        if hasattr(args, 'color'):
            self['color'] = args.color
        dictConfig(self.logging_dict())

        logger.info("Starting ldap2pg %s.", __version__)

        # File loading.
        filename, mode = self.find_filename(os.environ, args)
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

        V.alias(
            file_config.get('postgres', {}),
            'roles_blacklist_query', 'blacklist',
        )

        # Now close stdin. To make SASL non-interactive.
        if not self.get('debug'):
            sys.stdin.close()

        check_yaml_gotchas(file_config)
        self.warn_unknown_config(file_config)

        # Now merge all config sources.
        default_privileges = make_well_known_privileges()
        try:
            self.merge(file_config=file_config, environ=os.environ, args=args)
            postprocess_privilege_options(self, default_privileges)
        except ValueError as e:
            raise ConfigurationError("Failed to load configuration: %s" % (e,))

        logger.debug("Configuration loaded.")

    def merge(self, file_config, environ=os.environ, args=object()):
        for mapping in self.MAPPINGS:
            value = mapping.process(
                default=deepget(self, mapping.path),
                file_config=file_config,
                environ=environ,
                args=args,
            )
            deepset(self, mapping.path, value)

        if self['verbose'] is not None:
            self['verbosity'] = 'DEBUG' if self['verbose'] else 'INFO'

    def read(self, fo, name, mode):
        try:
            payload = yaml.safe_load(fo)
        except yaml.error.YAMLError as e:
            msg = "YAML error with %s: %s" % (name, e)
            raise ConfigurationError(msg)

        if payload is None:
            raise ConfigurationError("Configuration is empty.")
        if isinstance(payload, list):
            payload = dict(sync_map=payload)
        if not isinstance(payload, dict):
            raise ConfigurationError("Configuration file must be a mapping.")
        if 'sync_map' not in payload:
            raise ConfigurationError("sync_map configuration is required.")

        payload['world_readable'] = bool(mode & 0o077)
        return payload

    def logging_dict(self):
        formatter = 'verbose' if self['verbosity'] == 'DEBUG' else 'info'
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
                    'level': self['verbosity'],
                },
            },
        }

    def warn_unknown_config(self, config):
        known_keys = set([m.path for m in self.MAPPINGS] + ['world_readable'])

        for k in iter_deep_keys(config):
            if k.startswith('privileges'):
                continue

            if k not in known_keys:
                logger.warning("Unknown config entry: %s.", k)
