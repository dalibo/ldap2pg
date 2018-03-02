# coding: utf-8

from __future__ import unicode_literals

import logging
from itertools import chain

import psycopg2

from .acl import (
    AclItem,
    AclSet,
)
from .role import (
    Role,
    RoleOptions,
    RoleSet,
)
from .utils import (
    UserError,
    match,
    unicode,
)


logger = logging.getLogger(__name__)


class PostgresInspector(object):
    def __init__(self, psql=None, acls=None, roles_blacklist=None, **queries):
        self.psql = psql
        self.acls = acls or {}
        self.queries = queries
        self.roles_blacklist = roles_blacklist or []

    def format_roles_query(self, name='all_roles'):
        query = self.queries[name]
        if not query:
            logger.warn("Roles introspection disabled.")
            return

        if isinstance(query, list):
            return query

        row_cols = ['rolname'] + RoleOptions.SUPPORTED_COLUMNS
        row_cols = ['role.%s' % (r,) for r in row_cols]
        return query.format(options=', '.join(row_cols[1:]))

    # Processors map tuples from psycopg2 to object.

    def row1(self, rows):
        # Just single value row for e.g databases, managed_roles, owners, etc.
        for row in rows:
            yield row[0]

    def process_roles(self, rows):
        # all_roles query signatures: name, [members, [options ...]]
        for row in rows:
            yield Role.from_row(*row)

    def process_grants(self, acl, dbname, rows):
        # GRANT query signatures: schema, role, [<acltype options> ...]
        for row in rows:
            if len(row) < 2:
                fmt = "%s ACL's inspect query doesn't return role as column 2"
                raise UserError(fmt % (acl,))

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

    # is_*_managed check whether an object should be ignored from inspection.

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
            self.queries.get('all_roles', False) is None
            or role == 'public'
            or role in roles
        )

    def is_owner_managed(self, owner, owners):
        return owner is None or owner in owners

    def is_schema_managed(self, schema, managed_schemas):
        return schema is None or schema in managed_schemas

    def filter_roles(self, allroles, whitelist):
        managedroles = RoleSet()
        for role in list(allroles):
            pattern = match(role.name, self.roles_blacklist)
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

    # Fetchers implements the logic to inspect cluster. The tricky part is that
    # inspect queries have various format : None, plain YAML list, SQL query
    # with different signature for backward compatibility or simply to adapt
    # various situations.

    def fetch(self, psql, name_or_sql, processor=None):
        # Implement common management of customizable queries.

        if isinstance(name_or_sql, unicode):
            sql = self.queries.get(name_or_sql, name_or_sql)
        elif name_or_sql is None:
            # Disabled inspection
            return []
        else:
            # Should be a static list.
            sql = name_or_sql

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

    def fetch_roles(self):
        # Actually, fetch databases for dropping objects, all roles and managed
        # roles. That's the minimum de synchronize roles.

        with self.psql() as psql:
            databases = self.fetch(psql, 'databases', self.row1)
            pgallroles = RoleSet(self.fetch(
                psql, self.format_roles_query(), self.process_roles))
            if not self.queries.get('managed_roles'):
                pgmanagedroles = set([r.name for r in pgallroles])
            else:
                logger.debug("Listing managed roles.")
                pgmanagedroles = set(self.fetch(
                    psql, 'managed_roles', self.row1))
        return databases, pgallroles, pgmanagedroles

    def fetch_schemas(self, databases, managedroles=None):
        # Fetch schemas and owners. This is required to trigger ACL inspection.
        # Owners are associated with schema, even if globally defined.

        schemas = dict([(k, []) for k in databases])
        for dbname, psql in self.psql.itersessions(databases):
            logger.debug("Inspecting schemas in %s", dbname)
            schemas[dbname] = dict(self.fetch(
                psql, 'schemas', self.process_schemas))
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
                            owners = set(self.fetch(psql, 'owners', self.row1))
                    s_owners = owners
                # Only filter if managedroles are defined. This allow ACL only
                # mode.
                if managedroles:
                    s_owners = s_owners & managedroles
                else:
                    s_owners = s_owners - set(self.roles_blacklist)
                schemas[dbname][schema] = s_owners

        return schemas

    def fetch_grants(self, schemas, roles):
        # Loop all defined ACL to inspect grants.

        pgacls = AclSet()
        for name, acl in sorted(self.acls.items()):
            if not acl.inspect:
                logger.warn("Can't inspect ACL %s: query not defined.", acl)
                continue

            logger.debug("Searching GRANTs of ACL %s.", acl)
            for dbname, psql in self.psql.itersessions(schemas):
                rows = psql(acl.inspect)
                # Gather all owners in database for global ACL
                owners = set(chain(*schemas[dbname].values()))
                for aclitem in self.process_grants(name, dbname, rows):
                    if not self.is_aclitem_managed(
                            aclitem, schemas, roles, owners):
                        continue
                    logger.debug("Found GRANT %s.", aclitem)
                    pgacls.add(aclitem)

        return pgacls
