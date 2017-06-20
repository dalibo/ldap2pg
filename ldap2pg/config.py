from __future__ import unicode_literals

import logging
import os
from os import stat
import re

import yaml

from .utils import (
    deepget,
    deepset,
    deepupdate,
)


logger = logging.getLogger(__name__)


_auto_env = object()


class Mapping(object):
    """Fetch value from either file or env var."""

    def __init__(self, path, env=_auto_env, secret=False):
        self.path = path
        if env == _auto_env:
            env = path.upper().replace(':', '_')
        self.env = env
        if isinstance(secret, str):
            secret = re.compile(secret)
        self.secret = secret

    def process(self, default, file_config, environ):
        deny_secret = file_config.get('world_readable', True)
        try:
            if self.env:
                value = environ[self.env]
            else:
                raise KeyError()
        except KeyError:
            try:
                value = deepget(file_config, self.path)
            except KeyError:
                return default
            else:
                if hasattr(self.secret, 'search'):
                    secret = self.secret.search(value)
                else:
                    secret = self.secret

                if secret and deny_secret:
                    raise ValueError(
                        "Refuse to load secret from world readable file."
                    )

        return value


class Configuration(dict):
    DEFAULTS = {
        'ldap': {
            'host': '',
            'port': 389,
            'bind': None,
            'password': None,
            'base': '',
            'filter': '(objectClass=organizationalRole)',
        },
        'postgres': {
            'dsn': '',
            'blacklist': ['pg_*', 'postgres'],
        },
    }

    MAPPINGS = [
        Mapping('ldap:host'),
        Mapping('ldap:port'),
        Mapping('ldap:bind'),
        Mapping('ldap:password', secret=True),
        Mapping('ldap:base'),
        Mapping('ldap:filter'),
        Mapping(
            'postgres:dsn', env='PGDSN',
            secret=r'(?:password=|:[^/][^/].*@)',
        ),
        Mapping('postgres:blacklist', env=None),
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
            except PermissionError as e:
                logger.warn("Can't try %s: permission denied.", candidate)
            except FileNotFoundError as e:
                continue
        raise FileNotFoundError("No configuration file found")

    def load(self):
        # Main entry point for config loading. Most io should be done here.
        try:
            filename, mode = self.find_filename(environ=os.environ)
        except FileNotFoundError:
            logger.debug("No configuration file found.")
            file_config = {}
        else:
            logger.debug("Opening configuration file %s.", filename)
            with open(filename) as fo:
                file_config = self.read(fo, mode)

        self.merge(file_config=file_config, environ=os.environ)
        logger.debug("Configuration loaded.")

    def merge(self, file_config, environ=os.environ):
        self.update(file_config)

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
            raise ValueError("Configuration file must be a mapping")
        payload['world_readable'] = bool(mode & 0o044)
        return payload

    def update(self, other):
        deepupdate(self, other)
