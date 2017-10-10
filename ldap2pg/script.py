from __future__ import print_function
from __future__ import unicode_literals

import logging.config
import os
import pdb
import sys

import psycopg2

from . import ldap
from .config import Configuration, ConfigurationError
from .manager import SyncManager
from .psql import PSQL
from .utils import UserError


logger = logging.getLogger(__name__)


def wrapped_main(config=None):
    config = config or Configuration()
    config.load()

    logging_config = config.logging_dict()
    logging.config.dictConfig(logging_config)

    try:
        ldapconn = ldap.connect(**config['ldap'])
    except ldap.LDAPError as e:
        message = "Failed to connect to LDAP: %s" % (e,)
        raise ConfigurationError(message)

    if config.get('dry', True):
        logger.warn("Running in dry mode. Postgres will be untouched.")
    else:
        logger.warn("Running in real mode.")

    psql = PSQL(connstring=config['postgres']['dsn'])
    manager = SyncManager(
        ldapconn=ldapconn, psql=psql,
        acl_dict=config['acl_dict'],
        acl_aliases=config['acl_aliases'],
        blacklist=config['postgres']['blacklist'],
        roles_query=config['postgres']['roles_query'],
        dry=config['dry'],
    )
    try:
        sync_data = manager.inspect(
            syncmap=config['sync_map'])
    except psycopg2.OperationalError as e:
        message = "Failed to connect to Postgres: %s." % (str(e).strip(),)
        raise ConfigurationError(message)

    count = manager.sync(*sync_data)

    action = "Comparison" if config['dry'] else "Synchronization"
    logger.info("%s complete.", action)

    return int(count > 0) if config['check'] else 0


def main():
    debug = os.environ.get('DEBUG', '').lower() in {'1', 'y'}
    verbose = os.environ.get('VERBOSE', '').lower() in {'1', 'y'}

    config = Configuration()
    config['debug'] = debug
    config['verbose'] = debug or verbose
    config['color'] = sys.stderr.isatty()
    logging.config.dictConfig(config.logging_dict())
    logger.debug("Debug mode enabled.")

    try:
        exit(wrapped_main(config))
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
    logger = logging.getLogger(__package__)
    main()
