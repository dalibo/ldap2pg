from __future__ import print_function
from __future__ import unicode_literals

import logging.config
import os
import pdb
import sys

import ldap3
import psycopg2

from . import __version__
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


def wrapped_main(config=None):
    config = config or Configuration()
    config.load()

    logging_config = config.logging_dict()
    logging.config.dictConfig(logging_config)

    logger.info("Starting ldap2pg %s.", __version__)
    logger.debug("Debug mode enabled.")

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


def main():
    debug = os.environ.get('DEBUG', '').lower() in {'1', 'y'}
    verbose = os.environ.get('VERBOSE', '').lower() in {'1', 'y'}

    config = Configuration()
    config['verbose'] = debug or verbose
    config['color'] = sys.stderr.isatty()
    logging.config.dictConfig(config.logging_dict())

    try:
        wrapped_main(config)
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
