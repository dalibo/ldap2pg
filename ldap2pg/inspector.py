# coding: utf-8

from __future__ import unicode_literals

import logging
from itertools import chain
from textwrap import dedent

import psycopg2

from .privilege import Grant
from .privilege import Acl
from .role import (
    Role,
    RoleOptions,
    RoleSet,
)
from .utils import (
    Timer,
    UserError,
    match,
    unicode,
)


logger = logging.getLogger(__name__)


class PostgresInspector(object):
    def __init__(
            self, psql=None, privileges=None,
            shared_queries=None, **queries):
        self.psql = psql
        self.privileges = privileges or {}
        self.shared_queries = shared_queries or {}
        self.roles_blacklist = []
        self.queries = queries
        self.query_cache = {}
        self.timer = Timer()

    def format_roles_query(self, name='all_roles'):
        query = self.queries[name]
        if not query:
            logger.warning("Roles introspection disabled.")
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

    def process_grants(self, privilege, dbname, rows):
        # GRANT query signatures: schema, role, [<privilege options> ...]
        sql = str(privilege.grant_sql) + str(privilege.revoke_sql)
        schema_aware = '{schema}' in sql
        for row in rows:
            if len(row) < 2:
                fmt = "%s's inspect query doesn't return role as column 2"
                raise UserError(fmt % (privilege,))

            # Explicitly ignore schema on schema naive privilege.
            if not schema_aware and row[0]:
                row = (None,) + row[1:]

            yield Grant.from_row(privilege.name, dbname, *row)

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

    def is_grant_managed(self, grant, schemas, roles, all_owners):
        if not self.is_role_managed(grant.role, roles):
            return False

        dbname, schema = grant.dbname, grant.schema
        if not self.is_schema_managed(schema, schemas[dbname]):
            return False

        # Use all owners in database for schema-less privileges
        owners = all_owners if not schema else schemas[dbname][schema]
        if not self.is_owner_managed(grant.owner, owners):
            return False

        return True

    def is_role_managed(self, role, roles):
        return role not in self.roles_blacklist and (
            self.queries.get('all_roles', False) is None
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
                    "Ignoring role '%s'. Matches %r.", role.name, pattern)
                # Remove blacklisted role from allroles. Prefer to fail on
                # re-CREATE-ing it rather than even altering options of it.
                allroles.remove(role)
            elif role.name not in whitelist:
                logger.debug("May reuse role '%s'.", role.name)
            else:
                logger.debug("Managing role '%s' %s.", role.name, role.options)
                if role.members:
                    # Filter members to not revoke unmanaged roles.
                    role.members = list(set(role.members) & whitelist)
                    logger.debug(
                        "Role '%s' has members %s.",
                        role.name, ','.join(role.members),
                    )
                managedroles.add(role)

        if 'public' in whitelist:
            managedroles.add(Role(name='public'))

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
                with self.timer:
                    rows = psql(sql)

            if processor:
                rows = processor(rows)
            if not isinstance(rows, list):
                # Track time spent fetching data from Postgres. It's about 5%
                # on testing env.
                rows = list(self.timer.time_iter(rows))
            return rows
        except psycopg2.ProgrammingError as e:
            # Consider the query as user defined
            raise UserError(str(e))

    inspect_me = dedent("""\
    SELECT current_user, rolsuper
    FROM pg_catalog.pg_roles
    WHERE rolname = current_user;
    """)

    def fetch_me(self):
        with self.psql() as psql:
            return self.fetch(psql, self.inspect_me)[0]

    def fetch_roles(self):
        # Actually, fetch databases for dropping objects, all roles and managed
        # roles. That's the minimum de synchronize roles.

        with self.psql() as psql:
            databases = self.fetch(psql, 'databases', self.row1)
            pgallroles = RoleSet(self.fetch(
                psql, self.format_roles_query(), self.process_roles))
            if not self.queries.get('managed_roles'):
                # Legacy ldap2pg manages public, always. We keep this by
                # default as it is sound to manage public privileges.
                pgmanagedroles = set(['public'] + [r.name for r in pgallroles])
            else:
                logger.debug("Listing managed roles.")
                pgmanagedroles = set(self.fetch(
                    psql, 'managed_roles', self.row1))
        return databases, pgallroles, pgmanagedroles

    def fetch_roles_blacklist(self):
        return self.fetch(self.psql, 'roles_blacklist_query', self.row1)

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
        # Loop all defined privileges to inspect grants.

        pgacl = Acl()
        for name, privilege in sorted(self.privileges.items()):
            if not privilege.inspect:
                logger.warning(
                    "Can't inspect privilege %s: query not defined.",
                    privilege)
                continue

            logger.debug("Searching GRANTs of privilege %s.", privilege)
            for dbname, psql in self.psql.itersessions(schemas):
                if isinstance(privilege.inspect, dict):
                    rows = self.fetch_shared_query(
                        name=privilege.inspect['shared_query'],
                        keys=privilege.inspect['keys'],
                        dbname=dbname,
                        psql=psql,
                    )
                else:
                    with self.timer:
                        rows = list(psql(privilege.inspect))
                    logger.debug("Took %s.", self.timer.last_delta)

                # Gather all owners in database for global ACL
                owners = set(chain(*schemas[dbname].values()))
                grants = self.process_grants(privilege, dbname, rows)
                for grant in self.timer.time_iter(iter(grants)):
                    if self.is_grant_managed(grant, schemas, roles, owners):
                        logger.debug("Found GRANT %s.", grant)
                        pgacl.add(grant)

        self.query_cache.clear()

        return pgacl

    def fetch_shared_query(self, name, keys, dbname, psql):
        cache_key = '%s_%s' % (name, dbname)
        if cache_key not in self.query_cache:
            with self.timer:
                rows = list(psql(self.shared_queries[name]))
            logger.debug("Took %s.", self.timer.last_delta)
            # Fill the row cache.
            self.query_cache[cache_key] = rows
        else:
            logger.debug("Reusing shared query cache %s.", name)

        # Now filter row by key, removing key.
        return [
            r[1:] for r in self.query_cache[cache_key]
            if r[0] in keys
        ]
