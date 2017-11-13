from __future__ import unicode_literals

from fnmatch import fnmatch
import logging
from itertools import groupby

from .ldap import LDAPError, str2dn

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
    _, attributes = entry
    path = attribute.split('.')
    try:
        values = attributes[path[0]]
    except KeyError:
        raise ValueError("Unknown attribute %r" % (path[0],))
    path = path[1:]
    for value in values:
        if path:
            dn = str2dn(value)
            value = dict()
            for (type_, name, _), in dn:
                names = value.setdefault(type_, [])
                names.append(name)
            logger.debug("Parsed DN: %s", value)
            try:
                value = value[path[0]][0]
            except KeyError:
                raise ValueError("Unknown attribute %s" % (path[0],))

        yield value


class SyncManager(object):

    def __init__(
            self, ldapconn=None, psql=None, acl_dict=None, acl_aliases=None,
            blacklist=[], roles_query=';', dry=False):
        self.ldapconn = ldapconn
        self.psql = psql
        self.acl_dict = acl_dict or {}
        self.acl_aliases = acl_aliases or {}
        self._blacklist = blacklist
        self._roles_query = roles_query
        self.dry = dry

    def fetch_database_list(self, psql):
        select = """
        SELECT datname FROM pg_catalog.pg_database
        WHERE datallowconn IS TRUE ORDER BY 1;
        """.strip().replace(8 * ' ', '')
        for row in psql(select):
            yield row[0]

    def fetch_schema_list(self, psql):
        select = """
        SELECT nspname FROM pg_catalog.pg_namespace
        WHERE nspname NOT LIKE 'pg_%' ORDER BY 1;
        """.strip().replace(8 * ' ', '')
        for row in psql(select):
            yield row[0]

    def fetch_pg_roles(self, psql):
        if not self._roles_query:
            logger.warn("Roles introspection disabled.")
            return

        row_cols = ['rolname'] + list(RoleOptions.COLUMNS_MAP.values())
        row_cols = ['role.%s' % (r,) for r in row_cols]
        qry = self._roles_query.format(options=', '.join(row_cols[1:]))
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

    def process_pg_acl_items(self, acl, dbname, rows):
        for row in rows:
            try:
                schema, role, full = row
            except ValueError:
                schema, role, full = row + (True,)

            if match(role, self._blacklist):
                continue

            yield AclItem.from_row(acl, dbname, schema, role, full)

    def query_ldap(self, base, filter, attributes, scope):
        try:
            entries = self.ldapconn.search_s(
                base, scope, filter, attributes,
            )
        except LDAPError as e:
            message = "Failed to query LDAP: %s." % (e,)
            raise UserError(message)

        logger.debug('Got %d entries from LDAP.', len(entries))
        return entries

    def process_ldap_entry(self, entry, **kw):
        if 'names' in kw:
            names = kw['names']
            log_source = " from YAML"
        else:
            name_attribute = kw['name_attribute']
            names = get_ldap_attribute(entry, name_attribute)
            log_source = " from %s %s" % (entry[0], name_attribute)

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
                try:
                    for role in self.process_ldap_entry(entry=entry, **rule):
                        yield role
                except ValueError as e:
                    msg = "Failed to process %.32s: %s" % (entry, e,)
                    raise UserError(msg)

    def apply_grant_rules(self, grant, dbname=None, schema=None, entries=[]):
        for rule in grant:
            acl = rule.get('acl')

            database = rule.get('database', dbname)
            if database == '__all__':
                database = AclItem.ALL_DATABASES

            schema = rule.get('schema', schema)
            if schema == '__all__':
                schema = AclItem.ALL_SCHEMAS
            elif schema == '__any__':
                schema = None

            pattern = rule.get('role_match')

            for entry in entries:
                if 'roles' in rule:
                    roles = rule['roles']
                else:
                    try:
                        roles = get_ldap_attribute(
                            entry, rule['role_attribute'])
                    except ValueError as e:
                        msg = "Failed to process %.32s: %s" % (entry, e,)
                        raise UserError(msg)

                for role in roles:
                    role = role.lower()
                    if pattern and not fnmatch(role, pattern):
                        logger.debug(
                            "Don't grant %s to %s not matching %s",
                            acl, role, pattern,
                        )
                        continue
                    yield AclItem(acl, database, schema, role)

    def inspect(self, syncmap):
        logger.info("Inspecting Postgres...")
        with self.psql('postgres') as psql:
            databases = list(self.fetch_database_list(psql))
            rows = self.fetch_pg_roles(psql)
            pgroles = RoleSet(self.process_pg_roles(rows))

        schemas = {k: [] for k in databases}
        for dbname, psql in self.psql.itersessions(databases):
            schemas[dbname] = list(self.fetch_schema_list(psql))
            logger.debug(
                "Found schemas %s in %s.", ', '.join(schemas[dbname]), dbname,
            )

        # Inspect ACLs
        pgacls = AclSet()
        for name, acl in sorted(self.acl_dict.items()):
            if not acl.inspect:
                logger.warn("Can't inspect ACL %s: query not defined.", acl)
                continue

            logger.debug("Searching items of ACL %s.", acl)
            for dbname, psql in self.psql.itersessions(databases):
                rows = psql(acl.inspect)
                for aclitem in self.process_pg_acl_items(name, dbname, rows):
                    logger.debug("Found ACL item %s.", aclitem)
                    pgacls.add(aclitem)

        logger.debug("Postgres inspection done.")

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

            grant = mapping.get('grant', [])
            aclitems = self.apply_grant_rules(grant, dbname, schema, entries)
            for aclitem in aclitems:
                logger.debug("Found ACL item %s in LDAP.", aclitem)
                for realitem in aclitem.expandaliases(self.acl_aliases):
                    ldapacls.add(realitem)

        logger.debug("LDAP inspection completed. Post processing.")
        ldaproles.resolve_membership()
        ldapacls = AclSet(list(ldapacls.expanditems(schemas)))

        return databases, pgroles, pgacls, ldaproles, ldapacls

    def diff(self, pgroles=None, pgacls=set(), ldaproles=None, ldapacls=set()):
        pgroles = pgroles or RoleSet()
        ldaproles = ldaproles or RoleSet()

        # First, revoke spurious ACLs
        spurious = pgacls - ldapacls
        spurious = sorted(list(spurious))
        for aclname, aclitems in groupby(spurious, lambda i: i.acl):
            acl = self.acl_dict[aclname]
            if not acl.revoke_sql:
                logger.warn("Can't revoke ACL %s: query not defined.", acl)
                continue
            for aclitem in aclitems:
                yield acl.revoke(aclitem)

        # Then create missing roles
        missing = RoleSet(ldaproles - pgroles)
        for role in missing.flatten():
            for qry in role.create():
                yield qry

        # Now update existing roles options and memberships
        existing = pgroles & ldaproles
        pg_roles_index = pgroles.reindex()
        ldap_roles_index = ldaproles.reindex()
        for role in existing:
            my = pg_roles_index[role.name]
            its = ldap_roles_index[role.name]
            for qry in my.alter(its):
                yield qry

        # Don't forget to trash all spurious roles!
        spurious = RoleSet(pgroles - ldaproles)
        for role in reversed(list(spurious.flatten())):
            for qry in role.drop():
                yield qry

        # Finally, grant ACL when all roles are ok.
        missing = ldapacls - {a for a in pgacls if a.full}
        missing = sorted(list(missing))
        for aclname, aclitems in groupby(missing, lambda i: i.acl):
            acl = self.acl_dict[aclname]
            if not acl.grant_sql:
                logger.warn("Can't grant ACL %s: query not defined.", acl)
                continue
            for aclitem in aclitems:
                yield acl.grant(aclitem)

    def sync(self, databases, pgroles, pgacls, ldaproles, ldapacls):
        count = 0
        queries = self.diff(pgroles, pgacls, ldaproles, ldapacls)
        for query in expandqueries(queries, databases):
            with self.psql(query.dbname) as psql:
                count += 1
                if self.dry:
                    logger.info('Would ' + lower1(query.message))
                else:
                    logger.info(query.message)

                sql = psql.mogrify(*query.args).decode('UTF-8')
                if self.dry:
                    logger.debug("Would execute: %s", sql)
                else:
                    try:
                        psql(sql)
                    except Exception as e:
                        msg = "Error while executing SQL query:\n%s" % (e,)
                        raise UserError(msg)
        logger.debug("Generated %d querie(s).", count)

        if not count:
            logger.info("Nothing to do.")

        return count
