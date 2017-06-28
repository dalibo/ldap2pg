from __future__ import print_function
from __future__ import unicode_literals

from . import __version__

import logging.config
import os
import pdb
import sys

import ldap3
import psycopg2

from .config import Configuration, ConfigurationError
from .manager import RoleManager
from .utils import UserError


logger = logging.getLogger(__name__)


def create_ldap_connection(host, port, bind, password, **kw):
    logger.debug("Connecting to LDAP server %s:%s.", host, port)
    server = ldap3.Server(host, port, get_info=ldap3.ALL)
    return ldap3.Connection(server, bind, password, auto_bind=True)


def create_pg_connection(dsn):
    logger.debug("Connecting to PostgreSQL.")
    return psycopg2.connect(dsn)


def wrapped_main():
    config = Configuration()
    config.load()

    try:
        ldapconn = create_ldap_connection(**config['ldap'])
        pgconn = create_pg_connection(dsn=config['postgres']['dsn'])
    except ldap3.core.exceptions.LDAPExceptionError as e:
        message = "Failed to connect to LDAP: %s" % (e,)
        raise ConfigurationError(message)
    except psycopg2.OperationalError as e:
        message = "Failed to connect to Postgres: %s." % (str(e).strip(),)
        raise ConfigurationError(message)

    manager = RoleManager(
        ldapconn=ldapconn, pgconn=pgconn,
        blacklist=config['postgres']['blacklist'],
        dry=config['dry'],
    )
    manager.sync(map_=config['sync_map'])


class MultilineFormatter(logging.Formatter):
    def format(self, record):
        s = super(MultilineFormatter, self).format(record)
        if '\n' not in s:
            return s

        lines = s.splitlines()
        d = record.__dict__.copy()
        for i, line in enumerate(lines[1:]):
            record.message = line
            lines[1+i] = self._fmt % record.__dict__
        record.__dict__ = d

        return '\n'.join(lines)


class ColorFormatter(MultilineFormatter):

    _color_map = {
        logging.DEBUG: '37',
        logging.INFO: '1;39',
        logging.WARN: '96',
        logging.ERROR: '91',
        logging.CRITICAL: '1;91',
    }

    def format(self, record):
        lines = super(ColorFormatter, self).format(record)
        color = self._color_map.get(record.levelno, '39')
        lines = ''.join([
            '\033[0;%sm%s\033[0m' % (color, line)
            for line in lines.splitlines(True)
        ])
        return lines


def logging_dict(tty=True, debug=False):
    formatter_kwargs = {'class': __name__ + '.ColorFormatter'} if tty else {}
    return {
        'version': 1,
        'formatters': {
            'debug': dict(
                format='[%(name)-16s %(levelname)8s] %(message)s',
                **formatter_kwargs
            ),
            'info': dict(format='%(message)s', **formatter_kwargs),
        },
        'handlers': {'stderr': {
            '()': 'logging.StreamHandler',
            'formatter': 'debug' if debug else 'info',
        }},
        'root': {
            'level': 'WARNING',
            'handlers': ['stderr'],
        },
        'loggers': {
            'ldap2pg': {
                'level': 'DEBUG' if debug else 'INFO',
            },
        },
    }


def main():
    debug = os.environ.get('DEBUG', '').lower() in {'1', 'y'}
    logging_config = logging_dict(debug=debug, tty=sys.stderr.isatty())
    logging.config.dictConfig(logging_config)
    logger.info("Starting ldap2pg %s.", __version__)
    logger.debug("Debug mode enabled.")

    try:
        wrapped_main()
        exit(0)
    except pdb.bdb.BdbQuit:
        logger.info("Graceful exit from debugger.")
    except UserError as e:
        logger.critical("%s", e)
        exit(e.exit_code)
    except Exception:
        logger.exception('Unhandled error:')
        if debug and sys.stdout.isatty():
            logger.debug("Dropping in debugger.")
            pdb.post_mortem(sys.exc_info()[2])
        else:
            logger.error(
                "Please file an issue at "
                "https://github.com/dalibo/ldap2pg/issues with full log.",
            )
    exit(os.EX_SOFTWARE)


if '__main__' == __name__:  # pragma: no cover
    main()
