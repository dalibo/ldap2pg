from itertools import chain

from .psql import Query
from .utils import AllDatabases, UserError, unicode, make_group_map


class Acl(object):
    TYPES = {}
    itemfmt = "%(dbname)s.%(schema)s for %(owner)s"

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

    def grant(self, item):
        fmt = "Grant %(acl)s on " + self.itemfmt + " to %(role)s."
        return Query(
            fmt % item.__dict__,
            item.dbname,
            self.grant_sql.format(
                database='"%s"' % item.dbname,
                schema='"%s"' % item.schema,
                owner='"%s"' % item.owner,
                role='"%s"' % item.role,
            ),
        )

    def revoke(self, item):
        fmt = "Revoke %(acl)s on " + self.itemfmt + " from %(role)s."
        return Query(
            fmt % item.__dict__,
            item.dbname,
            self.revoke_sql.format(
                database='"%s"' % item.dbname,
                schema='"%s"' % item.schema,
                owner='"%s"' % item.owner,
                role='"%s"' % item.role,
            ),
        )


@Acl.register
class DatAcl(Acl):
    itemfmt = '%(dbname)s'

    def expanddb(self, item, databases):
        if item.dbname is AclItem.ALL_DATABASES:
            dbnames = databases.keys()
        else:
            dbnames = item.dbname

        for dbname in dbnames:
            yield item.copy(acl=self.name, dbname=dbname)

    def expand(self, item, databases):
        for exp in self.expanddb(item, databases):
            # inspect query will return AclItem with NULL schema, so ensure we
            # have schema None.
            exp.schema = None
            yield exp


@Acl.register
class GlobalDefAcl(DatAcl):
    itemfmt = '%(dbname)s for %(owner)s'

    def expand(self, item, databases):
        for exp in super(GlobalDefAcl, self).expand(item, databases):
            for schema in databases[exp.dbname]:
                for owner in databases[exp.dbname][schema]:
                    yield exp.copy(owner=owner)


@Acl.register
class NspAcl(DatAcl):
    itemfmt = '%(dbname)s.%(schema)s'

    def expandschema(self, item, databases):
        if item.schema is AclItem.ALL_SCHEMAS:
            try:
                schemas = databases[item.dbname]
            except KeyError:
                fmt = "Database %s does not exists or is not managed."
                raise UserError(fmt % (item.dbname))
        else:
            schemas = item.schema
        for schema in schemas:
            yield item.copy(acl=self.name, schema=schema)

    def expand(self, item, databases):
        for datexp in self.expanddb(item, databases):
            for nspexp in self.expandschema(datexp, databases):
                yield nspexp


@Acl.register
class DefAcl(NspAcl):
    itemfmt = '%(dbname)s.%(schema)s for %(owner)s'

    def expand(self, item, databases):
        for expand in super(DefAcl, self).expand(item, databases):
            for owner in databases[expand.dbname][expand.schema]:
                yield expand.copy(owner=owner)


class AclItem(object):
    ALL_DATABASES = AllDatabases()
    ALL_SCHEMAS = None

    @classmethod
    def from_row(cls, *args):
        return cls(*args)

    def __init__(self, acl, dbname=None, schema=None, role=None, full=True,
                 owner=None):
        self.acl = acl
        self.dbname = dbname
        self.schema = schema
        self.role = role
        self.full = full
        self.owner = owner

    def __lt__(self, other):
        return self.as_tuple() < other.as_tuple()

    def __str__(self):
        full_map = {None: 'n/a', True: 'full', False: 'partial'}
        fmt = (
            '%(acl)s on %(dbname)s.%(schema)s for %(owner)s'
            ' to %(role)s (%(full)s)'
        )
        return fmt % dict(
            self.__dict__,
            schema=self.schema or '*',
            owner=self.owner or '*',
            full=full_map[self.full],
        )

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __hash__(self):
        return hash(''.join(chain(*filter(None, self.as_tuple()))))

    def __eq__(self, other):
        return self.as_tuple() == other.as_tuple()

    def as_tuple(self):
        return (
            self.dbname or '', self.role, self.acl, self.schema, self.owner)

    def copy(self, **kw):
        return self.__class__(**dict(dict(
            acl=self.acl,
            role=self.role,
            dbname=self.dbname,
            schema=self.schema,
            full=self.full,
            owner=self.owner,
        ), **kw))


class AclSet(set):
    def expanditems(self, aliases, acl_dict, databases):
        for item in self:
            try:
                aclnames = aliases[item.acl]
            except KeyError:
                raise ValueError("Unknown ACL %s" % (item.acl,))

            for aclname in aclnames:
                try:
                    acl = acl_dict[aclname]
                except KeyError:
                    raise ValueError("Unknown ACL %s" % (aclname,))

                for expansion in acl.expand(item, databases):
                    yield expansion


def check_group_definitions(acls, groups):
    known = set(acls.keys()) | set(groups.keys())
    for name, children in groups.items():
        unknown = [c for c in children if c not in known]
        if unknown:
            msg = 'Unknown ACL %s in group %s' % (
                ', '.join(sorted(unknown)), name)
            raise ValueError(msg)


def process_definitions(acls):
    # Check and manage ACL and ACL group definitions in same namespace.
    groups = {}
    for k, v in sorted(acls.items()):
        if isinstance(v, list):
            groups[k] = v
            acls.pop(k)

    check_group_definitions(acls, groups)
    aliases = make_group_map(acls, groups)

    return acls, groups, aliases
