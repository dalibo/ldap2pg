from __future__ import unicode_literals

from fnmatch import fnmatch
import logging

import psycopg2
from psycopg2 import sql

logger = logging.getLogger(__name__)


class RoleManager(object):

    def __init__(self, ldapconn, pgconn, blacklist=[]):
        self.ldapconn = ldapconn
        self.pgconn = pgconn
        self.pgcursor = None
        self._blacklist = blacklist

    def __enter__(self):
        self.pgcursor = self.pgconn.cursor()

    def __exit__(self, *a):
        self.pgcursor.close()

    def blacklist(self, items):
        for i in items:
            for pattern in self._blacklist:
                if fnmatch(i, pattern):
                    break
            else:
                yield i

    def fetch_pg_roles(self):
        logger.debug("Querying PostgreSQL for existing roles.")
        self.pgcursor.execute(
            "SELECT rolname FROM pg_catalog.pg_roles",
        )
        payload = self.pgcursor.fetchall()
        return {r[0] for r in payload}

    def fetch_ldap_roles(self, base, query):
        logger.debug("Querying LDAP for wanted roles.")
        self.ldapconn.search(base, query, attributes=['*'])
        return {r.cn.value for r in self.ldapconn.entries}

    def create(self, role):
        logger.info("Creating new role %s.", role)
        self.pgcursor.execute(
            sql.SQL('CREATE ROLE {name} WITH LOGIN').format(
                name=psycopg2.sql.Identifier(role),
            )
        )
        self.pgconn.commit()

    def drop(self, role):
        logger.warn("Dropping existing role %s.", role)
        self.pgcursor.execute(
            sql.SQL('DROP ROLE {name}').format(
                name=psycopg2.sql.Identifier(role),
            )
        )
        self.pgconn.commit()

    def sync(self, base, query):
        with self:
            pgroles = self.fetch_pg_roles()
            pgroles = set(self.blacklist(pgroles))
            ldaproles = self.fetch_ldap_roles(base=base, query=query)

            missing = ldaproles - pgroles
            for role in missing:
                self.create(role)

            spurious = pgroles - ldaproles
            for role in spurious:
                self.drop(role)

        logger.info("Synchronization complete.")
