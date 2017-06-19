from __future__ import print_function
from __future__ import unicode_literals

from . import __version__

import logging
import os

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


def main():
    logging.basicConfig(
        level=logging.DEBUG,
        format='%(levelname)5.5s %(message)s'
    )
    logger.debug("Starting ldap2pg %s.", __version__)

    try:
        ldapconn = create_ldap_connection(
            host=os.environ['LDAP_HOST'],
            bind=os.environ['LDAP_BIND'],
            password=os.environ['LDAP_PASSWORD'],
        )
        pgconn = create_pg_connection(dsn=os.environ.get('PGDSN', ''))

        ldap_base = os.environ['LDAP_BASE']
        ldap_query = '(objectClass=organizationalRole)'

        manager = RoleManager(ldapconn=ldapconn, pgconn=pgconn)
        manager.sync(base=ldap_base, query=ldap_query)
    except Exception:
        logger.exception('Unhandled error:')
        exit(1)


if '__main__' == __name__:
    main()
