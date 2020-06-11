from __future__ import unicode_literals

import itertools
import logging

from .ldap import LDAPError, RDNError, get_attribute, lower_attributes
from .privilege import Acl
from .role import (
    CommentError,
    RoleOptions,
    RoleSet,
)
from .utils import UserError, decode_value, make_format_vars, match
from .psql import expandqueries


logger = logging.getLogger(__name__)


class SyncManager(object):
    def __init__(
            self, ldapconn=None, psql=None, inspector=None,
            privileges=None, privilege_aliases=None,
    ):
        self.ldapconn = ldapconn
        self.psql = psql
        self.inspector = inspector
        self.privileges = privileges or {}
        self.privilege_aliases = privilege_aliases or {}

    @property
    def roles_blacklist(self):
        try:
            return self.inspector.roles_blacklist
        except AttributeError:
            return []

    def _query_ldap(self, base, filter, attributes, scope):
        # Query directory returning a list of entries. An entry is a triplet
        # containing Distinguished name, attributes and joins.
        try:
            raw_entries = self.ldapconn.search_s(
                base, scope, filter, attributes,
            )
        except LDAPError as e:
            message = "Failed to query LDAP: %s." % (e,)
            raise UserError(message)

        logger.debug('Got %d entries from LDAP.', len(raw_entries))
        entries = []
        wanted_attribute_names = attributes
        for dn, attributes in raw_entries:
            if not dn:
                logger.debug("Discarding ref: %.40s.", attributes)
                continue

            attributes['dn'] = [dn]
            for n in wanted_attribute_names:
                attributes.setdefault(n, [])

            try:
                entry = decode_value((dn, attributes))
            except UnicodeDecodeError as e:
                message = "Failed to decode data from %r: %s." % (dn, e,)
                raise UserError(message)

            entries.append(lower_attributes(entry) + ({},))

        return entries

    def query_ldap(self, base, filter, attributes, joins, scope):
        logger.info(
            "Querying LDAP %.24s... %.12s...",
            base, filter.replace('\n', ''))
        entries = self._query_ldap(base, filter, attributes, scope)

        join_cache = {}
        for attr, join in joins.items():
            for dn, attrs, entry_joins in entries:
                for value in attrs[attr]:
                    # That would be nice to group all joins of one entry.
                    join_key = '%s/%s' % (attr, value)
                    join_entries = join_cache.get(join_key)
                    if join_entries is None:
                        join_query = dict(join, base=value)
                        try:
                            logger.info("Sub-querying LDAP %.24s...", value)
                            join_entries = self._query_ldap(**join_query)
                            join_cache[join_key] = join_entries
                        except UserError as e:
                            logger.warning('Ignoring %s: %s', value, e)
                            join_cache[join_key] = False
                            continue
                    if join_entries:
                        join_entries += entry_joins.get(attr, [])
                        entry_joins[attr] = join_entries

        return entries

    def inspect_ldap(self, syncmap):
        ldaproles = {}
        ldapacl = Acl()
        for mapping in syncmap:
            if mapping.get('description'):
                logger.info("%s", mapping['description'])

            role_rules = mapping.get('roles', [])
            grant_rules = mapping.get('grant', [])
            if 'ldap' in mapping:
                on_unexpected_dn = mapping['ldap'].pop(
                    'on_unexpected_dn', 'fail')
                entries = self.query_ldap(**mapping['ldap'])
                fields = set(itertools.chain(*[
                    rule.all_fields
                    for rule in
                    role_rules + grant_rules
                ]))
                log_source = 'in LDAP'
            else:
                entries = [('YAML', {}, {})]
                fields = []
                log_source = 'from YAML'
                on_unexpected_dn = 'fail'

            for entry in entries:
                vars_ = self.build_format_vars(
                    entry,
                    fields,
                    on_unexpected_dn=on_unexpected_dn,
                )

                for rule in role_rules:
                    try:
                        self.apply_role_rule(
                            rule, ldaproles, vars_, log_source)
                    except CommentError as e:
                        raise UserError.wrap("""\
                        An error occured while generating comment on role from
                        LDAP: %s Ensure the comment format ("%s") is consistent
                        with role name format.
                        """ % (e, rule.comment.formats[0]))
                for rule in grant_rules:
                    self.apply_grant_rule(rule, ldapacl, vars_, log_source)

        # Lazy apply of role options defaults
        roleset = RoleSet()
        for role in ldaproles.values():
            role.options.fill_with_defaults()
            if role.comment is None:
                role.comment = 'Managed by ldap2pg.'
            roleset.add(role)

        return roleset, ldapacl

    @staticmethod
    def build_format_vars(entry, fields, on_unexpected_dn):
        values = {}
        for field in fields:
            values.setdefault(field, [])
            for value in get_attribute(entry, field):
                if isinstance(value, RDNError):
                    msg = "Unexpected DN: %s" % value.dn
                    if 'ignore' == on_unexpected_dn:
                        continue
                    elif 'warn' == on_unexpected_dn:
                        logger.warning(msg)
                    else:
                        raise UserError(msg)
                else:
                    values[field].append(value)
        return make_format_vars(fields, entry[0], values)

    def apply_role_rule(self, rule, ldaproles, vars_, log_source):
        for role in rule.generate(vars_):
            pattern = match(role.name, self.roles_blacklist)
            if pattern:
                logger.debug(
                    "Ignoring role %s %s. Matches %s.",
                    role, log_source, pattern)
                continue

            if role in ldaproles:
                try:
                    role.merge(ldaproles[role])
                except ValueError:
                    msg = "Role %s redefined with different options." % (
                        role,)
                    raise UserError(msg)
            else:
                logger.debug("Want role %s %s.", role, log_source)
            ldaproles[role] = role

    def apply_grant_rule(self, rule, ldapacl, vars_, log_source):
        for grant in rule.generate(vars_):
            pattern = match(grant.role, self.roles_blacklist)
            if pattern:
                logger.debug(
                    "Ignoring grant on role %s %s. Matches %s.",
                    grant.role, log_source, pattern)
                continue
            logger.debug("Want GRANT %s %s.", grant, log_source)
            ldapacl.add(grant)

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

    def sync(self, syncmap):
        if not syncmap:
            logger.warning(
                "Empty synchronization map. All roles will be dropped!")

        logger.info("Inspecting roles in Postgres cluster...")
        self.inspector.roles_blacklist = self.inspector.fetch_roles_blacklist()
        me, issuper = self.inspector.fetch_me()
        if not match(me, self.roles_blacklist):
            self.inspector.roles_blacklist.append(me)

        if not issuper:
            logger.warning("Running ldap2pg as non superuser.")
            RoleOptions.filter_super_columns()

        databases, pgallroles, pgmanagedroles = self.inspector.fetch_roles()
        pgallroles, pgmanagedroles = self.inspector.filter_roles(
            pgallroles, pgmanagedroles)

        logger.debug("Postgres roles inspection done.")
        ldaproles, ldapacl = self.inspect_ldap(syncmap)
        logger.debug("LDAP inspection completed. Post processing.")
        try:
            ldaproles.resolve_membership()
        except ValueError as e:
            raise UserError(str(e))

        count = 0
        count += self.psql.run_queries(expandqueries(
            pgmanagedroles.diff(other=ldaproles, available=pgallroles),
            databases=databases))
        if self.privileges:
            logger.info("Inspecting GRANTs in Postgres cluster...")
            # Inject ldaproles in managed roles to avoid requerying roles.
            pgmanagedroles.update(ldaproles)
            if self.psql.dry and count:
                logger.warning(
                    "In dry mode, some owners aren't created, "
                    "their default privileges can't be determined.")
            schemas = self.inspector.fetch_schemas(databases, ldaproles)
            pgacl = self.inspector.fetch_grants(schemas, pgmanagedroles)
            ldapacl = self.postprocess_acl(ldapacl, schemas)
            count += self.psql.run_queries(expandqueries(
                pgacl.diff(ldapacl, self.privileges),
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
