from .psql import Query
from .utils import AllDatabases, unicode


class AllSchemas(object):
    # Simple object to represent schema wildcard.
    def __repr__(self):
        return '__ALL_SCHEMAS__'


class Acl(object):
    TYPES = {}

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
        return Query(
            "Grant %s." % (item,),
            item.dbname,
            self.grant_sql.format(
                database='"%s"' % item.dbname,
                schema='"%s"' % item.schema,
                role='"%s"' % item.role,
            ),
        )

    def revoke(self, item):
        return Query(
            "Revoke %s." % (item,),
            item.dbname,
            self.revoke_sql.format(
                database='"%s"' % item.dbname,
                schema='"%s"' % item.schema,
                role='"%s"' % item.role,
            ),
        )


@Acl.register
class DatAcl(Acl):
    def expanddb(self, item, databases):
        if item.dbname is AclItem.ALL_DATABASES:
            dbnames = databases.keys()
        else:
            dbnames = [item.dbname]

        for dbname in dbnames:
            yield item.copy(acl=self.name, dbname=dbname)

    expand = expanddb


@Acl.register
class NspAcl(DatAcl):
    def expandschema(self, item, databases):
        if item.schema is AclItem.ALL_SCHEMAS:
            schemas = databases[item.dbname]
        else:
            schemas = [item.schema]
        for schema in schemas:
            yield item.copy(acl=self.name, schema=schema)

    def expand(self, item, databases):
        for datexp in self.expanddb(item, databases):
            for nspexp in self.expandschema(datexp, databases):
                yield nspexp


class AclItem(object):
    ALL_DATABASES = AllDatabases()
    ALL_SCHEMAS = AllSchemas()

    @classmethod
    def from_row(cls, *args):
        return cls(*args)

    def __init__(self, acl, dbname=None, schema=None, role=None, full=True):
        self.acl = acl
        self.dbname = dbname
        self.schema = schema
        self.role = role
        self.full = full

    def __lt__(self, other):
        return self.as_tuple() < other.as_tuple()

    def __str__(self):
        return '%(acl)s on %(dbname)s.%(schema)s to %(role)s' % dict(
            self.__dict__,
            schema=self.schema or '*'
        )

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __hash__(self):
        return hash(self.as_tuple())

    def __eq__(self, other):
        return self.as_tuple() == other.as_tuple()

    def as_tuple(self):
        return (self.acl, self.dbname, self.schema, self.role)

    def copy(self, **kw):
        return self.__class__(**dict(dict(
            acl=self.acl,
            role=self.role,
            dbname=self.dbname,
            schema=self.schema,
            full=self.full,
        ), **kw))


class AclSet(set):
    def expanditems(self, aliases, acl_dict, databases):
        for item in self:
            for aclname in aliases[item.acl]:
                acl = acl_dict[aclname]
                for expansion in acl.expand(item, databases):
                    yield expansion
