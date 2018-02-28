from __future__ import unicode_literals

from fnmatch import fnmatch
import logging
from itertools import chain, groupby

import psycopg2

from .ldap import LDAPError, get_attribute, lower_attributes

from .acl import AclItem, AclSet
from .role import (
    Role,
    RoleOptions,
    RoleSet,
)
from .utils import UserError, decode_value, lower1, match
from .psql import expandqueries


logger = logging.getLogger(__name__)


class SyncManager(object):
    def __init__(
            self, ldapconn=None, psql=None, acl_dict=None, acl_aliases=None,
            blacklist=[],
            roles_query=None, owners_query=None, managed_roles_query=None,
            databases_query=None, schemas_query=None):
        self.ldapconn = ldapconn
        self.psql = psql
        self.acl_dict = acl_dict or {}
        self.acl_aliases = acl_aliases or {}
        self._blacklist = blacklist
        self._managed_roles_query = managed_roles_query
        self._databases_query = databases_query
        self._owners_query = owners_query
        self._roles_query = roles_query
        self._schemas_query = schemas_query

    def row1(self, rows):
        for row in rows:
            yield row[0]

    def pg_fetch(self, psql, sql, processor=None):
        # Implement common management of customizable queries

        # Disabled inspection
        if sql is None:
            return []

        try:
            if isinstance(sql, list):
                # Static inspection
                rows = sql[:]
                if rows and not isinstance(rows[0], (list, tuple)):
                    rows = [(v,) for v in rows]
            else:
                rows = psql(sql)

            if processor:
                rows = processor(rows)
            if not isinstance(rows, list):
                rows = list(rows)
            return rows
        except psycopg2.ProgrammingError as e:
            # Consider the query as user defined
            raise UserError(str(e))

    def format_roles_query(self, roles_query=None):
        roles_query = roles_query or self._roles_query
        if not roles_query:
            logger.warn("Roles introspection disabled.")
            return

        if isinstance(roles_query, list):
            return roles_query

        row_cols = ['rolname'] + RoleOptions.SUPPORTED_COLUMNS
        row_cols = ['role.%s' % (r,) for r in row_cols]
        return roles_query.format(options=', '.join(row_cols[1:]))

    def process_pg_roles(self, rows):
        for row in rows:
            yield Role.from_row(*row)

    def filter_roles(self, allroles, blacklist, whitelist):
        managedroles = RoleSet()
        for role in list(allroles):
            pattern = match(role.name, blacklist)
            if pattern:
                logger.debug(
                    "Ignoring role %s. Matches %r.", role.name, pattern)
                # Remove blacklisted role from allroles. Prefer to fail on
                # re-CREATE-ing it rather than even altering options of it.
                allroles.remove(role)
            elif role.name not in whitelist:
                logger.debug("May reuse role %s.", role.name)
            else:
                logger.debug("Managing role %r %s.", role.name, role.options)
                if role.members:
                    # Filter members to not revoke unmanaged roles.
                    role.members = list(set(role.members) & whitelist)
                    logger.debug(
                        "Role %s has members %s.",
                        role.name, ','.join(role.members),
                    )
                managedroles.add(role)

        return allroles, managedroles

    def process_pg_acl_items(self, acl, dbname, rows):
        for row in rows:
            try:
                role = row[1]
            except IndexError:
                fmt = "%s ACL's inspect query doesn't return role as column 2"
                raise UserError(fmt % (acl,))

            if match(role, self._blacklist):
                continue

            yield AclItem.from_row(acl, dbname, *row)

    def process_schemas(self, rows):
        for row in rows:
            if not isinstance(row, (list, tuple)):
                row = [row]

            # schemas_query can return tuple with signature (schema,) or
            # (schema, owners)
            try:
                schema, owners = row
                owners = owners or []
            except ValueError:
                schema, = row
                # Store that schemas_query is not aware of owners. e.g. static
                # list or old query.
                owners = False

            yield schema, owners

    def query_ldap(self, base, filter, attributes, scope):
        try:
            entries = self.ldapconn.search_s(
                base, scope, filter, attributes,
            )
        except LDAPError as e:
            message = "Failed to query LDAP: %s." % (e,)
            raise UserError(message)

        logger.debug('Got %d entries from LDAP.', len(entries))
        entries = decode_value(entries)
        return [lower_attributes(e) for e in entries]

    def process_ldap_entry(self, entry, **kw):
        if 'names' in kw:
            names = kw['names']
            log_source = " from YAML"
        else:
            name_attribute = kw['name_attribute']
            names = get_attribute(entry, name_attribute)
            log_source = " from %s %s" % (entry[0], name_attribute)

        if kw.get('members_attribute'):
            members = get_attribute(entry, kw['members_attribute'])
        else:
            members = []
        members = [m.lower() for m in members]

        kw.setdefault('parents', [])
        if kw.get('parents_attribute'):
            kw['parents'] += get_attribute(entry, kw['parents_attribute'])
        parents = [p.lower() for p in kw['parents']]

        for name in names:
            name = name.lower()
            logger.debug("Found role %s%s.", name, log_source)
            if members:
                logger.debug(
                    "Role %s must have members %s.", name, ', '.join(members),
                )
            if parents:
                logger.debug(
                    "Role %s is member of %s.", name, ', '.join(parents))
            role = Role(
                name=name,
                members=members,
                options=kw.get('options', {}),
                parents=parents[:],
            )

            yield role

    def apply_role_rules(self, rules, entries):
        for rule in rules:
            for entry in entries:
                try:
                    for role in self.process_ldap_entry(entry=entry, **rule):
                        yield role
                except ValueError as e:
                    msg = "Failed to process %.48s: %s" % (entry[0], e,)
                    raise UserError(msg)

    def apply_grant_rules(self, grant, entries=[]):
        for rule in grant:
            acl = rule.get('acl')

            databases = rule.get('databases', '__all__')
            if databases == '__all__':
                databases = AclItem.ALL_DATABASES

            schemas = rule.get('schemas', '__all__')
            if schemas in (None, '__all__', '__any__'):
                schemas = None

            pattern = rule.get('role_match')

            for entry in entries:
                if 'roles' in rule:
                    roles = rule['roles']
                else:
                    try:
                        roles = get_attribute(entry, rule['role_attribute'])
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
                    yield AclItem(acl, databases, schemas, role)

    def inspect_pg_roles(self):
        with self.psql() as psql:
            databases = self.pg_fetch(psql, self._databases_query, self.row1)
            pgallroles = RoleSet(self.pg_fetch(
                psql, self.format_roles_query(), self.process_pg_roles))
            if self._managed_roles_query is None:
                pgmanagedroles = set([r.name for r in pgallroles])
            else:
                logger.debug("Listing managed roles.")
                pgmanagedroles = set(self.pg_fetch(
                    psql, self._managed_roles_query, self.row1))
        return databases, pgallroles, pgmanagedroles

    def is_aclitem_managed(self, aclitem, schemas, roles, all_owners):
        if not self.is_role_managed(aclitem.role, roles):
            return False

        dbname, schema = aclitem.dbname, aclitem.schema
        if not self.is_schema_managed(schema, schemas[dbname]):
            return False

        # Use all owners in database for schema-less ACLs
        owners = all_owners if not schema else schemas[dbname][schema]
        if not self.is_owner_managed(aclitem.owner, owners):
            return False

        return True

    def is_role_managed(self, role, roles):
        return (
            self._roles_query is None
            or role == 'public'
            or role in roles
        )

    def is_owner_managed(self, owner, owners):
        return owner is None or owner in owners

    def is_schema_managed(self, schema, managed_schemas):
        return (
            schema is None
            or schema in managed_schemas
        )

    def inspect_schemas(self, databases, managedroles=None):
        schemas = dict([(k, []) for k in databases])
        for dbname, psql in self.psql.itersessions(databases):
            logger.debug("Inspecting schemas in %s", dbname)
            schemas[dbname] = dict(self.pg_fetch(
                psql, self._schemas_query, self.process_schemas))
            logger.debug(
                "Found schemas %s in %s.",
                ', '.join(schemas[dbname]), dbname)

        # Fallback to postgres:owners_query if schemas_query does not return
        # owners.
        owners = None
        for dbname in schemas:
            for schema in schemas[dbname]:
                if schemas[dbname][schema] is not False:
                    s_owners = set(schemas[dbname][schema])
                else:
                    # False owner means schemas_query is not aware of owners.
                    if owners is None:
                        logger.debug("Globally inspecting owners...")
                        with self.psql() as psql:
                            owners = set(self.pg_fetch(
                                    psql, self._owners_query, self.row1))
                    s_owners = owners
                # Only filter if managedroles are defined. This allow ACL only mode
                if managedroles:
                    s_owners = s_owners & managedroles
                else:
                    s_owners = s_owners - set(self._blacklist)
                schemas[dbname][schema] = s_owners

        return schemas

    def inspect_pg_acls(self, syncmap, schemas, roles):
        pgacls = AclSet()
        for name, acl in sorted(self.acl_dict.items()):
            if not acl.inspect:
                logger.warn("Can't inspect ACL %s: query not defined.", acl)
                continue

            logger.debug("Searching items of ACL %s.", acl)
            for dbname, psql in self.psql.itersessions(schemas):
                rows = psql(acl.inspect)
                # Gather all owners in database for global ACL
                owners = set(chain(*schemas[dbname].values()))
                for aclitem in self.process_pg_acl_items(name, dbname, rows):
                    if not self.is_aclitem_managed(
                            aclitem, schemas, roles, owners):
                        continue
                    logger.debug("Found ACL item %s.", aclitem)
                    pgacls.add(aclitem)

        return pgacls

    def inspect_ldap(self, syncmap):
        ldaproles = {}
        ldapacls = AclSet()
        for mapping in syncmap:
            if 'ldap' in mapping:
                logger.info(
                    "Querying LDAP %.24s... %.12s...",
                    mapping['ldap']['base'], mapping['ldap']['filter'])
                entries = self.query_ldap(**mapping['ldap'])
                log_source = 'in LDAP'
            else:
                entries = [None]
                log_source = 'from YAML'

            for role in self.apply_role_rules(mapping['roles'], entries):
                if role in ldaproles:
                    if role.options != ldaproles[role].options:
                        msg = "Role %s redefined with different options." % (
                            role,)
                        raise UserError(msg)
                    role.merge(ldaproles[role])
                ldaproles[role] = role

            grant = mapping.get('grant', [])
            aclitems = self.apply_grant_rules(grant, entries)
            for aclitem in aclitems:
                logger.debug("Found ACL item %s %s.", aclitem, log_source)
                ldapacls.add(aclitem)

        return RoleSet(ldaproles.values()), ldapacls

    def postprocess_acls(self, ldapacls, schemas):
        expanded_acls = ldapacls.expanditems(
            aliases=self.acl_aliases,
            acl_dict=self.acl_dict,
            databases=schemas,
        )

        ldapacls = AclSet()
        try:
            for aclitem in expanded_acls:
                ldapacls.add(aclitem)
        except ValueError as e:
            raise UserError(e)

        return ldapacls

    def diff_roles(self, pgallroles=None, pgmanagedroles=None, ldaproles=None):
        pgallroles = pgallroles or RoleSet()
        pgmanagedroles = pgmanagedroles or RoleSet()
        ldaproles = ldaproles or RoleSet()

        # First create missing roles
        missing = RoleSet(ldaproles - pgallroles)
        for role in missing.flatten():
            for qry in role.create():
                yield qry

        # Now update existing roles options and memberships
        existing = pgallroles & ldaproles
        pg_roles_index = pgallroles.reindex()
        ldap_roles_index = ldaproles.reindex()
        for role in existing:
            my = pg_roles_index[role.name]
            its = ldap_roles_index[role.name]
            if role not in pgmanagedroles:
                logger.warn(
                    "Role %s already exists in cluster. Reusing.", role.name)
            for qry in my.alter(its):
                yield qry

        # Don't forget to trash all spurious managed roles!
        spurious = RoleSet(pgmanagedroles - ldaproles)
        for role in reversed(list(spurious.flatten())):
            for qry in role.drop():
                yield qry

    def diff_acls(self, pgacls=None, ldapacls=None):
        pgacls = pgacls or AclSet()
        ldapacls = ldapacls or AclSet()

        # First, revoke spurious ACLs
        spurious = pgacls - ldapacls
        spurious = sorted([i for i in spurious if i.full is not None])
        for aclname, aclitems in groupby(spurious, lambda i: i.acl):
            acl = self.acl_dict[aclname]
            if not acl.revoke_sql:
                logger.warn("Can't revoke ACL %s: query not defined.", acl)
                continue
            for aclitem in aclitems:
                yield acl.revoke(aclitem)

        # Finally, grant ACL when all roles are ok.
        missing = ldapacls - set([a for a in pgacls if a.full in (None, True)])
        missing = sorted(list(missing))
        for aclname, aclitems in groupby(missing, lambda i: i.acl):
            acl = self.acl_dict[aclname]
            if not acl.grant_sql:
                logger.warn("Can't grant ACL %s: query not defined.", acl)
                continue
            for aclitem in aclitems:
                yield acl.grant(aclitem)

    def sync(self, syncmap):
        logger.info("Inspecting Postgres roles...")
        databases, pgallroles, pgmanagedroles = self.inspect_pg_roles()
        pgallroles, pgmanagedroles = self.filter_roles(
            pgallroles, self._blacklist, pgmanagedroles)

        logger.debug("Postgres inspection done.")
        ldaproles, ldapacls = self.inspect_ldap(syncmap)
        logger.debug("LDAP inspection completed. Post processing.")
        try:
            ldaproles.resolve_membership()
        except ValueError as e:
            raise UserError(str(e))

        count = 0
        count += self.psql.run_queries(expandqueries(
            self.diff_roles(pgallroles, pgmanagedroles, ldaproles),
            databases=databases))
        if self.acl_dict:
            logger.info("Inspecting Postgres ACLs...")
            if self.psql.dry and count:
                logger.warn(
                    "In dry mode, some owners aren't created, "
                    "their default privileges can't be determined.")
            schemas = self.inspect_schemas(databases, ldaproles)
            pgacls = self.inspect_pg_acls(syncmap, schemas, pgmanagedroles)
            ldapacls = self.postprocess_acls(ldapacls, schemas)
            count += self.psql.run_queries(expandqueries(
                self.diff_acls(pgacls, ldapacls),
                databases=schemas))
        else:
            logger.debug("No ACL defined. Skipping ACL. ")

        if count:
            # If log does not fit in 24 row screen, we should tell how much is
            # to be done.
            level = logger.debug if count < 20 else logger.info
            level("Generated %d querie(s).", count)
        else:
            logger.info("Nothing to do.")

        return count
