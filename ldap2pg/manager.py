from __future__ import unicode_literals

from fnmatch import fnmatch
import logging

from ldap3.core.exceptions import LDAPObjectClassError
from ldap3.utils.dn import parse_dn

from .utils import UserError


logger = logging.getLogger(__name__)


class Role(object):
    def __init__(self, name):
        self.name = name

    def __eq__(self, other):
        return self.name == str(other)

    def __hash__(self):
        return hash(self.name)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.name)

    def __str__(self):
        return self.name


class RoleSet(set):
    def diff(self, other):
        # Yields SQL queries to synchronize self with other.
        spurious = self - other
        for role in spurious:
            yield 'DROP ROLE %s;' % (role.name)
        missing = other - self
        for role in missing:
            yield 'CREATE ROLE %s;' % (role.name,)


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
                if fnmatch(str(i), pattern):
                    logger.debug("Ignoring role %s. Matches %r.", i, pattern)
                    break
            else:
                yield i

    def fetch_pg_roles(self):
        self.psql("SELECT rolname FROM pg_catalog.pg_roles;")
        payload = self.pgcursor.fetchall()
        return {Role(name=r[0]) for r in payload}

    def query_ldap(self, base, filter, attributes):
        logger.debug(
            "Doing: ldapsearch -h %s -p %s -D %s -W -b %s '%s' %s",
            self.ldapconn.server.host, self.ldapconn.server.port,
            self.ldapconn.user,
            base, filter, ' '.join(attributes or []),
        )
        self.ldapconn.search(base, filter, attributes=attributes)
        return self.ldapconn.entries[:]

    def process_ldap_entry(self, entry, name_attribute, **kw):
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
            yield Role(name=value)

    def psql(self, query):
        logger.debug("Doing: %s", query)
        self.pgcursor.execute(query)
        self.pgconn.commit()

    def sync(self, map_):
        with self:
            logger.info("Inspecting Postgres...")
            pgroles = self.fetch_pg_roles()
            pgroles = RoleSet(self.blacklist(pgroles))
            ldaproles = RoleSet()
            for mapping in map_:
                try:
                    logger.info("Querying LDAP...")
                    entries = self.query_ldap(**mapping['ldap'])
                except LDAPObjectClassError as e:
                    raise UserError("Failed to query LDAP: %s." % (e,))
                for entry in entries:
                    for rolmap in mapping['roles']:
                        roles = self.process_ldap_entry(
                            entry=entry, **rolmap
                        )
                        ldaproles |= set(roles)

            for query in pgroles.diff(ldaproles):
                logger.info("%s: %s", 'Would' if self.dry else 'Doing', query)
                if self.dry:
                    continue

                self.psql(query)
            else:
                logger.info("Nothing to do.")

        logger.info("Synchronization complete.")
        return ldaproles
