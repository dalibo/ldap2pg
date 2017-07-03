from __future__ import unicode_literals

from fnmatch import fnmatch
from itertools import groupby
import logging

from ldap3.core.exceptions import LDAPObjectClassError
from ldap3.utils.dn import parse_dn

from .role import (
    Role,
    RoleOptions,
    RoleSet,
)
from .utils import UserError, lower1


logger = logging.getLogger(__name__)


def get_ldap_attribute(entry, attribute):
    path = attribute.split('.')
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
        yield value


class RoleManager(object):

    def __init__(self, ldapconn=None, psql=None, blacklist=[], dry=False):
        self.ldapconn = ldapconn
        self.psql = psql
        self._blacklist = blacklist
        self.dry = dry

    # See https://www.postgresql.org/docs/current/static/view-pg-roles.html and
    # https://www.postgresql.org/docs/current/static/catalog-pg-auth-members.html
    _roles_query = """
    SELECT
        role.rolname, array_agg(members.rolname) AS members, %(options)s
    FROM
        pg_catalog.pg_roles AS role
    LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
    LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
    GROUP BY role.rolname, %(options)s
    ORDER BY 1;
    """.replace("\n    ", "\n").strip()

    def fetch_pg_roles(self, psql):
        row_cols = ['rolname'] + list(RoleOptions.COLUMNS_MAP.values())
        row_cols = ['role.%s' % (r,) for r in row_cols]
        qry = self._roles_query % dict(options=', '.join(row_cols[1:]))
        for row in psql(qry):
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
                logger.debug("Found role %r %s.", role.name, role.options)
                if role.members:
                    logger.debug(
                        "Role %s has members %s.",
                        role.name, ','.join(role.members),
                    )
                yield role

    def query_ldap(self, base, filter, attributes):
        logger.debug(
            "Doing: ldapsearch -h %s -p %s -D %s -W -b %s '%s' %s",
            self.ldapconn.server.host, self.ldapconn.server.port,
            self.ldapconn.user,
            base, filter, ' '.join(attributes or []),
        )

        try:
            self.ldapconn.search(base, filter, attributes=attributes)
        except LDAPObjectClassError as e:
            message = "Failed to query LDAP: %s." % (e,)
            raise UserError(message)

        return self.ldapconn.entries[:]

    def process_ldap_entry(self, entry, **kw):
        if 'names' in kw:
            names = kw['names']
            log_source = " from YAML"
        else:
            name_attribute = kw['name_attribute']
            names = get_ldap_attribute(entry, name_attribute)
            log_source = " from %s %s" % (entry.entry_dn, name_attribute)

        if kw.get('members_attribute'):
            members = get_ldap_attribute(entry, kw['members_attribute'])
            members = [m.lower() for m in members]
        else:
            members = []

        parents = [p.lower() for p in kw.get('parents', [])]

        for name in names:
            name = name.lower()
            logger.debug("Found role %s%s", name, log_source)
            if members:
                logger.debug(
                    "Role %s must have members %s.", name, ', '.join(members),
                )
            role = Role(
                name=name,
                members=members,
                options=kw.get('options', {}),
                parents=parents[:],
            )

            yield role

    def itermappings(self, syncmap):
        for dbname, schemas in syncmap.items():
            for schema, mappings in schemas.items():
                for mapping in mappings:
                    yield dbname, schema, mapping

    def apply_role_rules(self, rules, entries):
        for rule in rules:
            for entry in entries:
                for role in self.process_ldap_entry(entry=entry, **rule):
                    yield role

    def sync(self, map_):
        logger.info("Inspecting Postgres...")
        with self.psql('postgres') as psql:
            rows = self.fetch_pg_roles(psql)
            pgroles = RoleSet(self.process_pg_roles(rows))

        # Gather wanted roles
        ldaproles = RoleSet()
        for dbname, schema, mapping in self.itermappings(map_):
            logger.debug("Working on schema %s.%s.", dbname, schema)
            if 'ldap' in mapping:
                logger.info("Querying LDAP %s...", mapping['ldap']['base'])
                entries = self.query_ldap(**mapping['ldap'])
            else:
                entries = [None]

            roles = self.apply_role_rules(mapping['roles'], entries)
            ldaproles |= set(roles)
        ldaproles.resolve_membership()

        # Apply roles to Postgres
        queries = groupby(pgroles.diff(ldaproles), lambda q: q.dbname)
        count = 0
        for dbname, dbqueries in queries:
            with self.psql(dbname) as psql:
                logger.info("Synchronizing database %s.", dbname)
                for dbcount, query in enumerate(dbqueries):
                    count += 1
                    msg = str(query)
                    logger.info('Would ' + lower1(msg) if self.dry else msg)

                    sql = psql.mogrify(*query.args).decode('UTF-8')
                    if self.dry:
                        logger.debug("Would execute: %s", sql)
                    else:
                        psql(sql)
        logger.debug("Executed %d querie(s).", count)

        if not count:
            logger.info("Nothing to do.")
