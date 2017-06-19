from __future__ import print_function
from __future__ import unicode_literals

from . import __version__

import logging
import os

import ldap3
import psycopg2


logger = logging.getLogger(__name__)


def main():
    logging.basicConfig(
        level=logging.DEBUG,
        format='%(levelname).5s %(message)s'
    )
    logger.debug("Starting ldap2pg %s.", __version__)

    try:
        logger.debug("Connecting to LDAP.")
        server = ldap3.Server(os.environ['LDAP_HOST'], get_info=ldap3.ALL)
        conn = ldap3.Connection(
            server, os.environ['LDAP_BIND'], os.environ['LDAP_PASSWORD'],
            auto_bind=True,
        )

        logger.debug("Connecting to PostgreSQL from env vars.")
        psycopg2.connect(os.environ.get('PGDSN', ''))

        logger.debug("Searching LDAP.")
        conn.search(os.environ['LDAP_BASE'], '(objectClass=*)')
    except Exception:
        logger.exception('/o\\')
        exit(1)

    print(r'\o/')


if '__main__' == __name__:
    main()
