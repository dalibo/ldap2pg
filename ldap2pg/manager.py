from __future__ import unicode_literals

from fnmatch import fnmatch
import logging

from ldap3.core.exceptions import LDAPObjectClassError
from ldap3.utils.dn import parse_dn

from .role import (
    Role,
    RoleOptions,
    RoleSet,
)
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

    def fetch_pg_roles(self):
        self.psql(
            "SELECT rolname, %(cols)s FROM pg_catalog.pg_roles ORDER BY 1;"
            % dict(
                cols=', '.join(RoleOptions.COLUMNS_MAP.values()),
            )
        )
        for row in self.pgcursor:
            yield row

    def process_pg_roles(self, rows):
        for row in rows:
            name = row[0]
            for pattern in self._blacklist:
                if fnmatch(name, pattern):
                    logger.debug(
                        "Ignoring role %s. Matches %r.", name, pattern,
                    )
                    break
            else:
                role = Role.from_row(*row)
                logger.debug(
                    "Found role %r %s.", role.name, role.options,
                )
                yield role

    def query_ldap(self, base, filter, attributes):
        logger.debug(
            "Doing: ldapsearch -h %s -p %s -D %s -W -b %s '%s' %s",
            self.ldapconn.server.host, self.ldapconn.server.port,
            self.ldapconn.user,
            base, filter, ' '.join(attributes or []),
        )
        self.ldapconn.search(base, filter, attributes=attributes)
        return self.ldapconn.entries[:]

    def process_ldap_entry(self, entry, name_attribute, options=None, **kw):
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
                "Found role %s from %s %s",
                value, entry.entry_dn, name_attribute,
            )
            role = Role(name=value)
            if options:
                role.options.update(options)
            yield role

    def psql(self, query):
        logger.debug("Doing: %s", query)
        self.pgcursor.execute(query)
        self.pgconn.commit()

    def sync(self, map_):
        with self:
            logger.info("Inspecting Postgres...")
            rows = self.fetch_pg_roles()
            pgroles = RoleSet(self.process_pg_roles(rows))
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

            count = 0
            for query in pgroles.diff(ldaproles):
                count += 1
                logger.info("%s: %s", 'Would' if self.dry else 'Doing', query)
                if self.dry:
                    continue

                self.psql(query)

            if not count:
                logger.info("Nothing to do.")

        logger.info("Synchronization complete.")
        return ldaproles
