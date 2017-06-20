from __future__ import print_function
from __future__ import unicode_literals

from . import __version__

import logging
import os
import pdb
import sys

import ldap3
import psycopg2

from .manager import RoleManager


logger = logging.getLogger(__name__)


def create_ldap_connection(host, bind, password):
    logger.debug("Connecting to LDAP server %s.", host)
    server = ldap3.Server(host, get_info=ldap3.ALL)
    return ldap3.Connection(server, bind, password, auto_bind=True)


def create_pg_connection(dsn):
    logger.debug("Connecting to PostgreSQL.")
    return psycopg2.connect(dsn)


def wrapped_main():
    ldapconn = create_ldap_connection(
        host=os.environ['LDAP_HOST'],
        bind=os.environ['LDAP_BIND'],
        password=os.environ['LDAP_PASSWORD'],
    )
    pgconn = create_pg_connection(dsn=os.environ.get('PGDSN', ''))

    manager = RoleManager(
        ldapconn=ldapconn, pgconn=pgconn,
        blacklist=['pg_*', 'postgres'],
    )
    manager.sync(
        base=os.environ['LDAP_BASE'],
        query='(objectClass=organizationalRole)',
    )


def main():
    debug = os.environ.get('DEBUG', '').lower() in {'1', 'y'}
    logging.basicConfig(
        level=logging.DEBUG if debug else logging.INFO,
        format='%(levelname)5.5s %(message)s'
    )
    logger.debug("Starting ldap2pg %s.", __version__)

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
    exit(1)


if '__main__' == __name__:  # pragma: no cover
    main()
