from __future__ import print_function
from __future__ import unicode_literals

from . import __version__

import logging
import os
import pdb
import sys

import ldap3
import psycopg2

from .config import Configuration
from .manager import RoleManager


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

    ldapconn = create_ldap_connection(**config['ldap'])
    pgconn = create_pg_connection(dsn=config['postgres']['dsn'])

    manager = RoleManager(
        ldapconn=ldapconn, pgconn=pgconn,
        blacklist=config['postgres']['blacklist'],
        dry=config['dry'],
    )
    manager.sync(map_=config['sync_map'])


def main():
    debug = os.environ.get('DEBUG', '').lower() in {'1', 'y'}
    logging.basicConfig(
        level=logging.DEBUG if debug else logging.INFO,
        format='%(levelname)5.5s %(message)s'
    )
    logger.info("Starting ldap2pg %s.", __version__)

    try:
        wrapped_main()
        exit(0)
    except pdb.bdb.BdbQuit:
        logger.info("Graceful exit from debugger.")
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
    exit(1)


if '__main__' == __name__:  # pragma: no cover
    main()
