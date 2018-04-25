from __future__ import unicode_literals

from itertools import groupby
from fnmatch import fnmatch
import logging

from .ldap import LDAPError, get_attribute, lower_attributes

from .privilege import Grant
from .privilege import Acl
from .role import (
    Role,
    RoleSet,
)
from .utils import UserError, decode_value
from .psql import expandqueries


logger = logging.getLogger(__name__)


class SyncManager(object):
    def __init__(
            self, ldapconn=None, psql=None, inspector=None,
            privileges=None, privilege_aliases=None, blacklist=None,
    ):
        self.ldapconn = ldapconn
        self.psql = psql
        self.inspector = inspector
        self.privileges = privileges or {}
        self.privilege_aliases = privilege_aliases or {}
        self._blacklist = blacklist

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

        members = kw.get('members', [])[:]
        if kw.get('members_attribute'):
            members += get_attribute(entry, kw['members_attribute'])
        members = [m.lower() for m in members]

        parents = kw.get('parents', [])[:]
        if kw.get('parents_attribute'):
            parents += get_attribute(entry, kw['parents_attribute'])
        parents = [p.lower() for p in parents]

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
            privilege = rule.get('privilege')

            databases = rule.get('databases', '__all__')
            if databases == '__all__':
                databases = Grant.ALL_DATABASES

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
                            privilege, role, pattern,
                        )
                        continue
                    yield Grant(privilege, databases, schemas, role)

    def inspect_ldap(self, syncmap):
        ldaproles = {}
        ldapacl = Acl()
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
                    try:
                        role.merge(ldaproles[role])
                    except ValueError as e:
                        msg = "Role %s redefined with different options." % (
                            role,)
                        raise UserError(msg)
                ldaproles[role] = role

            grant = mapping.get('grant', [])
            grants = self.apply_grant_rules(grant, entries)
            for grant in grants:
                logger.debug("Found GRANT %s %s.", grant, log_source)
                ldapacl.add(grant)

        # Lazy apply of role options defaults
        roleset = RoleSet()
        for role in ldaproles.values():
            role.options.fill_with_defaults()
            roleset.add(role)

        return roleset, ldapacl

    def postprocess_acl(self, acl, schemas):
        expanded_grants = acl.expandgrants(
            aliases=self.privilege_aliases,
            privileges=self.privileges,
            databases=schemas,
        )

        acl = Acl()
        try:
            for grant in expanded_grants:
                acl.add(grant)
        except ValueError as e:
            raise UserError(e)

        return acl

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

    def diff_acls(self, pgacl=None, ldapacl=None):
        pgacl = pgacl or Acl()
        ldapacl = ldapacl or Acl()

        # First, revoke spurious GRANTs
        spurious = pgacl - ldapacl
        spurious = sorted([i for i in spurious if i.full is not None])
        for priv, grants in groupby(spurious, lambda i: i.privilege):
            acl = self.privileges[priv]
            if not acl.revoke_sql:
                logger.warn("Can't revoke %s: query not defined.", acl)
                continue
            for grant in grants:
                yield acl.revoke(grant)

        # Finally, grant privilege when all roles are ok.
        missing = ldapacl - set([a for a in pgacl if a.full in (None, True)])
        missing = sorted(list(missing))
        for priv, grants in groupby(missing, lambda i: i.privilege):
            priv = self.privileges[priv]
            if not priv.grant_sql:
                logger.warn("Can't grant %s: query not defined.", priv)
                continue
            for grant in grants:
                yield priv.grant(grant)

    def sync(self, syncmap):
        logger.info("Inspecting roles in Postgres cluster...")
        databases, pgallroles, pgmanagedroles = self.inspector.fetch_roles()
        pgallroles, pgmanagedroles = self.inspector.filter_roles(
            pgallroles, pgmanagedroles)

        logger.debug("Postgres inspection done.")
        ldaproles, ldapacl = self.inspect_ldap(syncmap)
        logger.debug("LDAP inspection completed. Post processing.")
        try:
            ldaproles.resolve_membership()
        except ValueError as e:
            raise UserError(str(e))

        count = 0
        count += self.psql.run_queries(expandqueries(
            self.diff_roles(pgallroles, pgmanagedroles, ldaproles),
            databases=databases))
        if self.privileges:
            logger.info("Inspecting GRANTs in Postgres cluster...")
            if self.psql.dry and count:
                logger.warn(
                    "In dry mode, some owners aren't created, "
                    "their default privileges can't be determined.")
            schemas = self.inspector.fetch_schemas(databases, ldaproles)
            pgacl = self.inspector.fetch_grants(schemas, pgmanagedroles)
            ldapacl = self.postprocess_acl(ldapacl, schemas)
            count += self.psql.run_queries(expandqueries(
                self.diff_acls(pgacl, ldapacl),
                databases=schemas))
        else:
            logger.debug("No privileges defined. Skipping GRANT and REVOKE.")

        if count:
            # If log does not fit in 24 row screen, we should tell how much is
            # to be done.
            level = logger.debug if count < 20 else logger.info
            level("Generated %d querie(s).", count)
        else:
            logger.info("Nothing to do.")

        return count
