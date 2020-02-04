from __future__ import print_function
from __future__ import unicode_literals

import logging
import os
import pdb
import resource
import sys

import psycopg2

from . import ldap
from .config import Configuration, ConfigurationError, dictConfig
from .inspector import PostgresInspector
from .manager import SyncManager
from .psql import PSQL
from .utils import UserError
from .role import RoleOptions


logger = logging.getLogger(__name__)


def wrapped_main(config=None):
    config = config or Configuration()
    config.load()

    logging_config = config.logging_dict()
    dictConfig(logging_config)

    if config.has_ldap_query():
        logger.debug("Setting up LDAP client.")
        try:
            ldapconn = ldap.connect(**config['ldap'])
        except ldap.LDAPError as e:
            message = "Failed to connect to LDAP: %s" % (e,)
            raise ConfigurationError(message)
    else:
        ldapconn = None

    if config.get('dry', True):
        logger.warning("Running in dry mode. Postgres will be untouched.")
    else:
        logger.info("Running in real mode.")
    psql = PSQL(connstring=config['postgres']['dsn'], dry=config['dry'])
    try:
        with psql() as psql_:
            supported_columns = psql_(RoleOptions.COLUMNS_QUERY).fetchone()[0]
    except psycopg2.OperationalError as e:
        message = "Failed to connect to Postgres: %s." % (str(e).strip(),)
        raise ConfigurationError(message)
    RoleOptions.update_supported_columns(supported_columns)

    inspector = PostgresInspector(
        psql=psql,
        privileges=config['privileges'],
        databases=config['postgres']['databases_query'],
        schemas=config['postgres']['schemas_query'],
        all_roles=config['postgres']['roles_query'],
        managed_roles=config['postgres']['managed_roles_query'],
        owners=config['postgres']['owners_query'],
        roles_blacklist_query=config['postgres']['roles_blacklist_query'],
        shared_queries=config['postgres']['shared_queries'],
    )
    manager = SyncManager(
        ldapconn=ldapconn, psql=psql, inspector=inspector,
        privileges=config['privileges'],
        privilege_aliases=config['privilege_aliases'],
    )
    count = manager.sync(syncmap=config['sync_map'])

    action = "Comparison" if config['dry'] else "Synchronization"
    logger.info("%s complete.", action)

    logger.debug("Inspecting Postgres took %s.", inspector.timer.delta)
    if ldapconn:
        logger.debug("Searching directory took %s.", ldapconn.timer.delta)
    logger.debug("Synchronizing Postgres took %s.", psql.timer.delta)

    rusage = resource.getrusage(resource.RUSAGE_SELF)
    logger.debug("Used up to %.1fMiB of memory.", rusage.ru_maxrss / 1024.)

    return int(count > 0) if config['check'] else 0


def main():
    config = Configuration()
    debug = False

    try:
        debug = config.bootstrap(environ=os.environ)
        if debug:
            logger.debug("Debug mode enabled.")
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
