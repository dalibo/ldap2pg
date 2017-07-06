from __future__ import unicode_literals

import logging

from ldap3.core.exceptions import LDAPExceptionError
from ldap3.utils.dn import parse_dn

from .acl import AclItem, AclSet
from .role import (
    Role,
    RoleOptions,
    RoleSet,
)
from .utils import UserError, lower1, match
from .psql import expandqueries


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

    def __init__(
            self, ldapconn=None, psql=None, acl_dict=None, blacklist=[],
            dry=False):
        self.ldapconn = ldapconn
        self.psql = psql
        self.acl_dict = acl_dict or {}
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

    def fetch_database_list(self, psql):
        select = """
        SELECT datname FROM pg_catalog.pg_database
        WHERE datallowconn IS TRUE ORDER BY 1;
        """.strip().replace(8 * ' ', '')
        for row in psql(select):
            yield row[0]

    def fetch_pg_roles(self, psql):
        row_cols = ['rolname'] + list(RoleOptions.COLUMNS_MAP.values())
        row_cols = ['role.%s' % (r,) for r in row_cols]
        qry = self._roles_query % dict(options=', '.join(row_cols[1:]))
        for row in psql(qry):
            yield row

    def process_pg_roles(self, rows):
        for row in rows:
            name = row[0]
            pattern = match(name, self._blacklist)
            if pattern:
                logger.debug("Ignoring role %s. Matches %r.", name, pattern)
                continue
            else:
                role = Role.from_row(*row)
                logger.debug("Found role %r %s.", role.name, role.options)
                if role.members:
                    logger.debug(
                        "Role %s has members %s.",
                        role.name, ','.join(role.members),
                    )
                yield role

    def process_pg_acl_items(self, name, rows):
        for row in rows:
            if match(row[2], self._blacklist):
                continue
            yield AclItem.from_row(name, *row)

    def query_ldap(self, base, filter, attributes):
        logger.debug(
            "Doing: ldapsearch -h %s -p %s -D %s -W -b %s '%s' %s",
            self.ldapconn.server.host, self.ldapconn.server.port,
            self.ldapconn.user,
            base, filter, ' '.join(attributes or []),
        )

        try:
            self.ldapconn.search(base, filter, attributes=attributes)
        except LDAPExceptionError as e:
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
            logger.debug("Found role %s%s.", name, log_source)
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

    def apply_grant_rules(self, grant, entries):
        acl = grant.get('acl')
        if not acl:
            return

        database = grant['database']
        if database == '__common__':
            raise ValueError("You must associate an ACL to a database.")
        schema = grant['schema']
        if schema == '__common__':
            schema = None

        for entry in entries:
            for role in get_ldap_attribute(entry, grant['role_attribute']):
                yield AclItem(acl, database, schema, role)

    def inspect(self, syncmap):
        logger.info("Inspecting Postgres...")
        with self.psql('postgres') as psql:
            databases = list(self.fetch_database_list(psql))
            rows = self.fetch_pg_roles(psql)
            pgroles = RoleSet(self.process_pg_roles(rows))

        # Inspect ACLs
        pgacls = AclSet()
        for name, acl in sorted(self.acl_dict.items()):
            for dbname, psql in self.psql.itersessions(databases):
                logger.debug("Searching items of ACL %s in %s.", acl, dbname)
                rows = psql(acl.inspect)
                for aclitem in self.process_pg_acl_items(name, rows):
                    logger.debug("Found ACL item %s.", aclitem)
                    pgacls.add(aclitem)

        # Gather wanted roles
        ldaproles = RoleSet()
        ldapacls = AclSet()
        for dbname, schema, mapping in self.itermappings(syncmap):
            logger.debug("Working on schema %s.%s.", dbname, schema)
            if 'ldap' in mapping:
                logger.info("Querying LDAP %s...", mapping['ldap']['base'])
                entries = self.query_ldap(**mapping['ldap'])
            else:
                entries = [None]

            for role in self.apply_role_rules(mapping['roles'], entries):
                ldaproles.add(role)

            grant = mapping.get('grant', {})
            grant.setdefault('database', dbname)
            grant.setdefault('schema', schema)
            for aclitem in self.apply_grant_rules(grant, entries):
                logger.debug("Found ACL item %s in LDAP.", aclitem)
                ldapacls.add(aclitem)

        logger.debug("LDAP inspection completed. Resolving memberships.")
        ldaproles.resolve_membership()

        return databases, pgroles, pgacls, ldaproles, ldapacls

    def sync(self, databases, pgroles, pgacls, ldaproles, ldapacls):
        count = 0
        queries = pgroles.diff(ldaproles)
        for query in expandqueries(queries, databases):
            with self.psql(query.dbname) as psql:
                count += 1
                msg = str(query)
                logger.info('Would ' + lower1(msg) if self.dry else msg)

                sql = psql.mogrify(*query.args).decode('UTF-8')
                if self.dry:
                    logger.debug("Would execute: %s", sql)
                else:
                    psql(sql)
        logger.debug("Generated %d querie(s).", count)

        if not count:
            logger.info("Nothing to do.")
