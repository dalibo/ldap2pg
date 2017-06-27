from __future__ import unicode_literals

from fnmatch import fnmatch
import logging

from ldap3.core.exceptions import LDAPObjectClassError
from ldap3.utils.dn import parse_dn

from .utils import UserError


logger = logging.getLogger(__name__)


class RoleManager(object):

    def __init__(self, ldapconn, pgconn, blacklist=[], dry=False):
        self.ldapconn = ldapconn
        self.pgconn = pgconn
        self.pgcursor = None
        self._blacklist = blacklist
        self.dry = dry

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

    def query_ldap(self, base, filter, attributes):
        logger.debug("Querying LDAP...")
        self.ldapconn.search(
            base, filter, attributes=attributes,
        )
        return self.ldapconn.entries[:]

    def process_ldap_entry(self, entry, name_attribute):
        path = name_attribute.split('.')
        values = entry.entry_attributes_as_dict[path[0]]
        path = path[1:]
        for value in values:
            if path:
                dn = parse_dn(value)
                value = dict()
                for type_, name, _ in dn:
                    names = value.setdefault(type_, [])
                    names.append(name)
                logger.debug("Parsed DN: %s", value)
                value = value[path[0]][0]
            logger.debug(
                "Yielding role %s from %s %s",
                value, entry.entry_dn, name_attribute,
            )
            yield value

    def create(self, role):
        if self.dry:
            return logger.info("Would create role %s.", role)

        logger.info("Creating new role %s.", role)
        self.pgcursor.execute('CREATE ROLE %s WITH LOGIN' % (role,))
        self.pgconn.commit()

    def drop(self, role):
        if self.dry:
            return logger.warn("Would drop role %s.", role)

        logger.warn("Dropping existing role %s.", role)
        self.pgcursor.execute('DROP ROLE %s' % (role,))
        self.pgconn.commit()

    def sync(self, map_):
        with self:
            pgroles = self.fetch_pg_roles()
            pgroles = set(self.blacklist(pgroles))
            ldaproles = set()
            for mapping in map_:
                try:
                    entries = self.query_ldap(**mapping['ldap'])
                except LDAPObjectClassError as e:
                    raise UserError("Failed to query LDAP: %s." % (e,))
                for entry in entries:
                    for rolmap in mapping['roles']:
                        roles = self.process_ldap_entry(
                            entry=entry, **rolmap
                        )
                        ldaproles |= set(roles)

            missing = ldaproles - pgroles
            for role in missing:
                self.create(role)

            spurious = pgroles - ldaproles
            for role in spurious:
                self.drop(role)

        logger.info("Synchronization complete.")
        return ldaproles
