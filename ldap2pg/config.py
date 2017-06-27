from __future__ import unicode_literals

import errno
import logging
import os
from os import stat
import re

from six import string_types
import yaml

from .utils import (
    deepget,
    deepset,
    UserError,
)


logger = logging.getLogger(__name__)


def raw(v):
    return v


def syncmap(value):
    if isinstance(value, dict):
        value = [value]

    if not value:
        raise ValueError("Empty mapping.")

    for item in value:
        item['ldap'] = dict(
            Configuration.DEFAULTS['ldap']['default_query'],
            **item['ldap']
        )
        ldap = item['ldap']
        if 'attribute' in ldap:
            ldap['attributes'] = ldap['attribute']
            del ldap['attribute']
        if isinstance(ldap['attributes'], str):
            ldap['attributes'] = [ldap['attributes']]

        if 'role' in item:
            item['roles'] = [item['role']]

        if 'roles' not in item:
            raise ValueError("Missing roles entry.")

    return value


_auto_env = object()


class Mapping(object):
    """Fetch value from either file or env var."""

    def __init__(self, path, env=_auto_env, secret=False, processor=raw):
        self.path = path
        if env == _auto_env:
            env = path.upper().replace(':', '_')
        self.env = env
        self.processor = processor
        if isinstance(secret, string_types):
            secret = re.compile(secret)
        self.secret = secret

    def process(self, default, file_config, environ):
        deny_secret = file_config.get('world_readable', True)
        try:
            if self.env:
                value = environ[self.env]
                logger.debug("Loaded %s from %s.", self.path, self.env)
            else:
                raise KeyError()
        except KeyError:
            try:
                value = deepget(file_config, self.path)
            except KeyError:
                value = default
            else:
                if hasattr(self.secret, 'search'):
                    secret = self.secret.search(value)
                else:
                    secret = self.secret

                if secret and deny_secret:
                    raise ValueError(
                        "Refuse to load secret from world readable file."
                    )

        return self.processor(value)


class ConfigurationError(UserError):
    def __init__(self, message):
        super(ConfigurationError, self).__init__(
            message, exit_code=os.EX_CONFIG,
        )


class NoConfigurationError(Exception):
    pass


class Configuration(dict):
    DEFAULTS = {
        'dry': False,
        'ldap': {
            'host': '',
            'port': 389,
            'bind': None,
            'password': None,
            'default_query': {
                'base': '',
                'filter': '(objectClass=organizationalRole)',
                'attributes': ['cn'],
            },
        },
        'postgres': {
            'dsn': '',
            'blacklist': ['pg_*', 'postgres'],
        },
        'sync_map': [],
    }

    MAPPINGS = [
        Mapping('dry'),
        Mapping('ldap:host'),
        Mapping('ldap:port'),
        Mapping('ldap:bind'),
        Mapping('ldap:password', secret=True),
        Mapping(
            'postgres:dsn', env='PGDSN',
            secret=r'(?:password=|:[^/][^/].*@)',
        ),
        Mapping('postgres:blacklist', env=None),
        Mapping('sync_map', env=None, processor=syncmap)
    ]

    def __init__(self):
        super(Configuration, self).__init__(self.DEFAULTS)

    _file_candidates = [
        './ldap2pg.yml',
        '~/.config/lda2pg.yml',
        '/etc/ldap2pg.yml',
    ]

    def find_filename(self, environ=os.environ):
        envval = environ.get('LDAP2PG_CONFIG')
        if envval:
            candidates = [envval]
        else:
            candidates = self._file_candidates

        for candidate in candidates:
            candidate = os.path.expanduser(candidate)
            try:
                logger.debug("Trying %s.", candidate)
                stat_ = stat(candidate)
                return candidate, stat_.st_mode
            except OSError as e:
                if e.errno == errno.EACCES:
                    logger.warn("Can't try %s: permission denied.", candidate)
        raise NoConfigurationError("No configuration file found")

    def load(self):
        # Main entry point for config loading. Most io should be done here.
        try:
            filename, mode = self.find_filename(environ=os.environ)
        except NoConfigurationError:
            logger.debug("No configuration file found.")
            file_config = {}
        else:
            logger.debug("Opening configuration file %s.", filename)
            with open(filename) as fo:
                file_config = self.read(fo, mode)

        try:
            self.merge(file_config=file_config, environ=os.environ)
        except ValueError as e:
            raise ConfigurationError("Failed to load configuration: %s" % (e,))

        logger.debug("Configuration loaded.")

    def merge(self, file_config, environ=os.environ):
        for mapping in self.MAPPINGS:
            value = mapping.process(
                default=deepget(self.DEFAULTS, mapping.path),
                file_config=file_config,
                environ=environ,
            )
            deepset(self, mapping.path, value)

    def read(self, fo, mode):
        payload = yaml.load(fo) or {}
        if not isinstance(payload, dict):
            raise ConfigurationError("Configuration file must be a mapping")
        payload['world_readable'] = bool(mode & 0o044)
        return payload
