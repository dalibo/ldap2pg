# coding: utf-8

from __future__ import unicode_literals

import logging
from itertools import chain
from textwrap import dedent

import psycopg2

from .privilege import Grant
from .privilege import Acl
from .psql import Query, scalar
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
            self, pool=None, privileges=None,
            shared_queries=None, **queries):
        self.pool = pool
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

    def process_grants(self, privilege, dbname, rows):
        # GRANT query signatures: schema, role, [<privilege options> ...]
        sql = str(privilege.grant_sql) + str(privilege.revoke_sql)
        schema_aware = '{schema}' in sql
        for row in handle_decoding_error(rows):
            if len(row) < 2:
                fmt = "%s's inspect query doesn't return role as column 2"
                raise UserError(fmt % (privilege,))

            # Explicitly ignore schema on schema naive privilege.
            if not schema_aware and row[0]:
                row = (None,) + row[1:]

            yield Grant.from_row(privilege.name, dbname, *row)

    # is_*_managed check whether an object should be ignored from inspection.

    def is_grant_managed(self, grant, db, roles):
        if not self.is_role_managed(grant.role, roles):
            return False

        if not self.is_schema_managed(grant.schema, db.schemas):
            return False

        # Use all owners in database for schema-less privileges
        owners = db.schemas[grant.schema].owners if grant.schema else db.owners
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
                continue
            elif role.name not in whitelist:
                logger.debug("May reuse role '%s'.", role.name)
            else:
                logger.debug("Managing role '%s' %s.", role.name, role.options)
                managedroles.add(role)

            if role.members:
                # Filter members to not revoke unmanaged roles.
                role.members = list(set(role.members) & whitelist)
                logger.debug(
                    "Role '%s' has members %s.",
                    role.name, ','.join(role.members),
                )

        if 'public' in whitelist:
            managedroles.add(Role(name='public'))

        return allroles, managedroles

    # Fetchers implements the logic to inspect cluster. The tricky part is that
    # inspect queries have various format : None, plain YAML list, SQL query
    # with different signature for backward compatibility or simply to adapt
    # various situations.

    def fetch(self, name_or_sql, row_factory=None, dbname=None):
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
                if row_factory:
                    rows = [row_factory(*r) for r in rows]
            else:
                conn = self.pool.getconn(dbname)
                with self.timer:
                    rows = conn.query(row_factory, sql)

            if not isinstance(rows, list):
                # Track time spent fetching data from Postgres. It's about 5%
                # on testing env.
                rows = list(self.timer.time_iter(handle_decoding_error(rows)))
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
        logger.debug("Introspecting session Postgres role.")
        return self.pool.getconn().queryone(None, self.inspect_me)

    inspect_databases = dedent("""\
    SELECT datname, rolname
    FROM pg_catalog.pg_database
    JOIN pg_catalog.pg_roles
      ON pg_catalog.pg_roles.oid = datdba
    WHERE datallowconn;
    """)

    def fetch_databases(self):
        logger.debug("Inspecting databases.")
        all_databases = self.fetch(self.inspect_databases, Database)
        managed_databases = self.fetch('databases', scalar)

        # Filter managed databases.
        all_databases = [
            db for db in all_databases if db.name in managed_databases
        ]

        return all_databases

    def fetch_roles(self):
        logger.debug("Inspecting all defined roles in cluster.")
        pgallroles = RoleSet(
            self.fetch(self.format_roles_query(), Role.from_row))
        if not self.queries.get('managed_roles'):
            # Legacy ldap2pg manages public, always. We keep this by
            # default as it is sound to manage public privileges.
            pgmanagedroles = set(['public'] + [r.name for r in pgallroles])
        else:
            logger.debug("Listing managed roles.")
            pgmanagedroles = set(self.fetch('managed_roles', scalar))
        return pgallroles, pgmanagedroles

    def fetch_roles_blacklist(self):
        return self.fetch('roles_blacklist_query', scalar)

    def fetch_schemas(self, databases, managedroles=None):
        # Fetch schemas and owners. This is required to trigger privilege
        # inspection. Owners are associated with schema, even if globally
        # defined.

        global_owners = None
        for db in databases:
            logger.debug("Inspecting schemas in %s", db)
            db.schemas = dict([
                (s.name, s) for s in self.fetch('schemas', Schema, db.name)
            ])
            logger.debug(
                "Found schemas %s in %s.",
                ', '.join(db.schemas.keys()), db)

            for schema in db.schemas.values():
                if schema.owners is False:
                    # Lazy inspect global owners from postgres:owners_query if
                    # schemas_query does not return owners. False owner means
                    # schemas_query is not aware of owners.
                    if global_owners is None:
                        logger.debug("Globally inspecting owners...")
                        global_owners = set(self.fetch('owners', scalar))
                    s_owners = global_owners
                else:
                    s_owners = set(schema.owners)

                # Only filter if managedroles are defined. This allow privilege
                # only mode.
                if managedroles:
                    s_owners = s_owners & managedroles
                else:
                    s_owners = s_owners - set(self.roles_blacklist)
                schema.owners = s_owners

        return databases

    def fetch_grants(self, databases, roles):
        # Loop all defined privileges to inspect grants.

        pgacl = Acl()
        for name, privilege in sorted(self.privileges.items()):
            if not privilege.inspect:
                logger.warning(
                    "Can't inspect privilege %s: query not defined.",
                    privilege)
                continue

            for db in databases:
                conn = self.pool.getconn(db.name)
                logger.debug(
                    "Searching GRANTs of privilege %s in %s.", privilege, db)
                if isinstance(privilege.inspect, dict):
                    rows = self.fetch_shared_query(
                        conn=conn,
                        name=privilege.inspect['shared_query'],
                        keys=privilege.inspect['keys'],
                        dbname=db.name,
                    )
                else:
                    with self.timer:
                        rows = list(conn.query(None, privilege.inspect))
                    logger.debug("Took %s.", self.timer.last_delta)

                # Gather all owners in database for global ACL
                grants = self.process_grants(privilege, db.name, rows)
                for grant in self.timer.time_iter(iter(grants)):
                    if self.is_grant_managed(grant, db, roles):
                        logger.debug("Found GRANT %s.", grant)
                        pgacl.add(grant)

        self.query_cache.clear()

        return pgacl

    def fetch_shared_query(self, name, keys, dbname, conn):
        cache_key = '%s_%s' % (name, dbname)
        if cache_key not in self.query_cache:
            with self.timer:
                rows = list(conn.query(None, self.shared_queries[name]))
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


class Database(object):
    def __init__(self, name, owner, managed=True):
        self.name = name
        self.owner = owner
        self.managed = managed
        self.schemas = {}

    def __eq__(self, other):
        return self.name == str(other)

    def __hash__(self):
        return hash(self.name)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.name)

    def __str__(self):
        return self.name

    @property
    def owners(self):
        # Object owners, while self.owner is Database owner.
        return set(chain(*[s.owners for s in self.schemas.values()]))

    def reassign(self, new_owner):
        yield Query(
            "Reassign database %s from %s to %s." % (
                self.name, self.owner, new_owner),
            None,
            """ALTER DATABASE "%s" OWNER TO "%s";""" % (
                self.name, new_owner,
            )
        )


class Schema(object):
    def __init__(self, name, owners=False):
        self.name = name
        self.owners = owners

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.name)

    def __str__(self):
        return self.name


def handle_decoding_error(iterator):
    try:
        for item in iterator:
            yield item
    except UnicodeDecodeError as e:
        raise UserError(
            "Encoding error: Can't decode %r from %s."
            % (e.object, e.encoding))
