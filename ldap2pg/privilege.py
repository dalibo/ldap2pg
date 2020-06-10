from itertools import groupby
import logging
from fnmatch import fnmatch

from .psql import Query
from .utils import (
    AllDatabases, FormatList, UserError,
    collect_fields, unicode, make_group_map,
)


logger = logging.getLogger(__name__)


class Privilege(object):
    TYPES = {}
    grantfmt = "%(dbname)s.%(schema)s for %(owner)s"

    def __init__(self, name, inspect=None, grant=None, revoke=None):
        self.name = name
        self.inspect = inspect
        self.grant_sql = grant
        self.revoke_sql = revoke

    def __eq__(self, other):
        return unicode(self) == unicode(other)

    def __lt__(self, other):
        return unicode(self) < unicode(other)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __str__(self):
        return self.name

    @classmethod
    def factory(cls, name, **kw):
        implcls = cls.TYPES[kw.pop('type')]
        return implcls(name, **kw)

    @classmethod
    def register(cls, subclass):
        cls.TYPES[subclass.__name__.lower()] = subclass
        return subclass

    def grant(self, grant):
        fmt = "Grant %(privilege)s on " + self.grantfmt + " to %(role)s."
        return Query(
            fmt % grant.as_dict(),
            grant.dbname,
            self.grant_sql.format(
                database='"%s"' % grant.dbname,
                schema='"%s"' % grant.schema,
                owner='"%s"' % grant.owner,
                role='"%s"' % grant.role,
            ),
        )

    def revoke(self, grant):
        fmt = "Revoke %(privilege)s on " + self.grantfmt + " from %(role)s."
        return Query(
            fmt % grant.as_dict(),
            grant.dbname,
            self.revoke_sql.format(
                database='"%s"' % grant.dbname,
                schema='"%s"' % grant.schema,
                owner='"%s"' % grant.owner,
                role='"%s"' % grant.role,
            ),
        )


@Privilege.register
class DatAcl(Privilege):
    grantfmt = '%(dbname)s'

    def expanddb(self, grant, databases):
        if grant.dbname is Grant.ALL_DATABASES:
            dbnames = databases.keys()
        else:
            dbnames = grant.dbname

        for dbname in dbnames:
            yield grant.copy(privilege=self.name, dbname=dbname)

    def expand(self, grant, databases):
        for exp in self.expanddb(grant, databases):
            # inspect query will return Grant with NULL schema, so ensure we
            # have schema None.
            exp.schema = None
            yield exp


@Privilege.register
class GlobalDefAcl(DatAcl):
    grantfmt = '%(dbname)s for %(owner)s'

    def expand(self, grant, databases):
        for exp in super(GlobalDefAcl, self).expand(grant, databases):
            for schema in databases[exp.dbname]:
                for owner in databases[exp.dbname][schema]:
                    yield exp.copy(owner=owner)


@Privilege.register
class NspAcl(DatAcl):
    grantfmt = '%(dbname)s.%(schema)s'

    def expandschema(self, grant, databases):
        if grant.schema is Grant.ALL_SCHEMAS:
            try:
                schemas = databases[grant.dbname]
            except KeyError:
                fmt = "Database %s does not exists or is not managed."
                raise UserError(fmt % (grant.dbname))
        else:
            schemas = grant.schema
        for schema in schemas:
            yield grant.copy(privilege=self.name, schema=schema)

    def expand(self, grant, databases):
        for datexp in self.expanddb(grant, databases):
            for nspexp in self.expandschema(datexp, databases):
                yield nspexp


@Privilege.register
class DefAcl(NspAcl):
    grantfmt = '%(dbname)s.%(schema)s for %(owner)s'

    def expand(self, grant, databases):
        for expand in super(DefAcl, self).expand(grant, databases):
            try:
                owners = databases[expand.dbname][expand.schema]
            except KeyError:
                msg = "Unknown schema %s.%s." % (
                    expand.dbname, expand.schema)
                raise UserError(msg)
            for owner in owners:
                yield expand.copy(owner=owner)


class Grant(object):
    ALL_DATABASES = AllDatabases()
    ALL_SCHEMAS = None

    __slots__ = (
        'dbname',
        'full',
        'owner',
        'privilege',
        'role',
        'schema',
    )

    @classmethod
    def from_row(cls, *args):
        return cls(*args)

    def __init__(
            self, privilege, dbname=None, schema=None, role=None, full=True,
            owner=None):
        self.privilege = privilege
        self.dbname = dbname
        self.schema = schema
        self.role = role
        self.full = full
        self.owner = owner

    def __lt__(self, other):
        return self.as_tuple() < other.as_tuple()

    _full_map = {None: 'n/a', True: 'full', False: 'partial'}

    def __str__(self):
        fmt = (
            '%s on %s.%s for %s'
            ' to %s (%s)'
        )
        dbname = self.dbname or '*'
        if isinstance(dbname, list):
            dbname = ','.join(dbname)
        schema = self.schema or '*'
        if isinstance(schema, list):
            schema = ','.join(schema)
        return fmt % (
            self.privilege,
            dbname,
            schema,
            self.owner or '*',
            self.role,
            self._full_map[self.full],
        )

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __hash__(self):
        return hash(tuple([unicode(x) for x in self.as_tuple()]))

    def __eq__(self, other):
        return self.as_tuple() == other.as_tuple()

    def as_dict(self):
        return dict(
            privilege=self.privilege,
            role=self.role,
            dbname=self.dbname,
            schema=self.schema,
            full=self.full,
            owner=self.owner,
        )

    def as_tuple(self):
        return (
            self.dbname or '', self.role, self.privilege, self.schema,
            self.owner)

    def copy(self, **kw):
        return self.__class__(**dict(self.as_dict(), **kw))


class GrantRule(object):
    def __init__(self, privilege, databases, schemas, roles, role_match=None):
        self.privilege = FormatList.factory([privilege])
        self.databases = FormatList.factory(databases or [])
        self.schemas = FormatList.factory(schemas or [])
        self.roles = FormatList.factory(roles)
        self.role_match = role_match
        self.all_fields = collect_fields(
            self.privilege, self.databases, self.schemas, self.roles,
        )

    def __repr__(self):
        return '<%s %s on [%s].[%s] to [%s]>' % (
            self.__class__.__name__,
            self.privilege,
            self.databases,
            self.schemas,
            self.roles,
        )

    def generate(self, vars_):
        privilege = next(self.privilege.expand(vars_))
        databases = list(self.databases.expand(vars_))
        if ['__all__'] == databases:
            databases = Grant.ALL_DATABASES

        schemas = list(self.schemas.expand(vars_))
        if not len(schemas):
            schemas = [None]
        if 1 == len(schemas) and schemas[0] in (None, '__all__', '__any__'):
            schemas = None

        try:
            roles = list(self.roles.expand(vars_))
        except ValueError as e:
            msg = "Failed to process %.32s: %s" % (vars_['dn'][0], e,)
            raise UserError(msg)

        for role in roles:
            if self.role_match and not fnmatch(role, self.role_match):
                logger.debug(
                    "Don't grant %s to %s not matching %s.",
                    privilege, role, self.role_match,
                )
                continue
            yield Grant(privilege, databases, schemas, role)

    def as_dict(self):
        dict_ = {}
        dict_['privilege'] = self.privilege.formats[0]
        for k in "databases", "schemas", "roles":
            dict_[k] = getattr(self, k).formats
        return dict_


class Acl(set):
    def expandgrants(self, aliases, privileges, databases):
        for grant in self:
            try:
                privnames = aliases[grant.privilege]
            except KeyError:
                raise ValueError("Unknown privilege %s" % (grant.privilege,))

            for name in privnames:
                try:
                    priv = privileges[name]
                except KeyError:
                    raise ValueError("Unknown privilege %s" % (name,))

                for expansion in priv.expand(grant, databases):
                    yield expansion

    def diff(self, other=None, privileges=None):
        # Yields query to match other from self.
        other = other or Acl()
        privileges = privileges or {}

        # First, revoke spurious GRANTs
        spurious = self - other
        spurious = sorted([i for i in spurious if i.full is not None])
        for priv, grants in groupby(spurious, lambda i: i.privilege):
            acl = privileges[priv]
            if not acl.revoke_sql:
                logger.warning("Can't revoke %s: query not defined.", acl)
                continue
            for grant in grants:
                yield acl.revoke(grant)

        # Finally, grant privilege when all roles are ok.
        missing = other - set([a for a in self if a.full in (None, True)])
        missing = sorted(list(missing))
        for priv, grants in groupby(missing, lambda i: i.privilege):
            priv = privileges[priv]
            if not priv.grant_sql:
                logger.warning("Can't grant %s: query not defined.", priv)
                continue
            for grant in grants:
                yield priv.grant(grant)


def check_group_definitions(privileges, groups):
    known = set(privileges.keys()) | set(groups.keys())
    for name, children in groups.items():
        unknown = [c for c in children if c not in known]
        if unknown:
            msg = 'Unknown privilege %s in group %s' % (
                ', '.join(sorted(unknown)), name)
            raise ValueError(msg)


def process_definitions(privileges):
    # Check and manage privileges and privilege groups definitions in same
    # namespace.
    groups = {}
    for k, v in sorted(privileges.items()):
        if isinstance(v, list):
            groups[k] = v
            privileges.pop(k)

    check_group_definitions(privileges, groups)
    aliases = make_group_map(privileges, groups)

    return privileges, groups, aliases
