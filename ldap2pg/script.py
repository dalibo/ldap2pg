from __future__ import print_function
from __future__ import unicode_literals

import logging
import os
import pdb
import resource
import sys
try:
    from StringIO import StringIO
except ImportError:
    from io import StringIO

import psycopg2

from . import ldap
from .config import Configuration, ConfigurationError
from .inspector import PostgresInspector
from .manager import SyncManager
from .psql import PSQL
from .utils import UserError
from .role import RoleOptions


logger = logging.getLogger(__name__)


def main():
    config = Configuration()
    debug = False

    try:
        debug = config.bootstrap(environ=os.environ)
        if debug:
            logger.debug("Debug mode enabled.")
        config.load()
        exit(synchronize(config))
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


def init_config(config, environ, argv):
    config_obj = Configuration()

    if isinstance(config, str):
        fo = StringIO(config)
        config = config_obj.read(fo, "string")
    else:
        config = config_obj.validate_raw_yaml(config, "dict")

    args = config_obj.read_argv(argv)
    config_obj.merge(config, environ, args)
    return config_obj


def synchronize(config=None, environ=None, argv=None):
    """Synchronize a Postgres cluster from LDAP directory

    This is the main entrypoint of ldap2pg logic. config is either a raw YAML
    string or a Python dict describing the ldap2pg configuration as documented
    in the YAML format.

    environ is a dict of environment variables, defaulting to os.environ. argv
    is the list of arguments passed to argparse and defaults to sys.argv[1:].

    If config['check'] is True, the return value is the number of queries
    generated to synchronize the cluster.

    In case of error, this procedure raises ldap2pg.UserError exception.

    """

    if not isinstance(config, Configuration):
        config = init_config(config, environ, argv)

    if config.has_ldapsearch():
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
            logger.debug("Inspecting role attributes.")
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


if '__main__' == __name__:  # pragma: no cover
    logger = logging.getLogger(__package__)
    main()
