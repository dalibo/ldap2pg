from .psql import Query
from .utils import AllDatabases, unicode


class AllSchemas(object):
    # Simple object to represent schema wildcard.
    def __repr__(self):
        return '__ALL_SCHEMAS__'


class Acl(object):
    def __init__(self, name, inspect=None, grant=None, revoke=None):
        self.name = name
        self.inspect = inspect
        self.grant_sql = grant
        self.revoke_sql = revoke

    def __lt__(self, other):
        return unicode(self) < unicode(other)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __str__(self):
        return self.name

    def grant(self, item):
        return Query(
            "Grant %s." % (item,),
            item.dbname,
            self.grant_sql.format(
                database=item.dbname,
                schema=item.schema,
                role=item.role,
            ),
        )

    def revoke(self, item):
        return Query(
            "Revoke %s." % (item,),
            item.dbname,
            self.revoke_sql.format(
                database=item.dbname,
                schema=item.schema,
                role=item.role,
            ),
        )


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

    def expandaliases(self, aliases):
        for acl in aliases[self.acl]:
            yield self.__class__(
                acl,
                self.dbname, self.schema, self.role,
                self.full,
            )

    def expand(self, databases):
        if self.dbname is self.ALL_DATABASES:
            dbnames = databases.keys()
        else:
            dbnames = [self.dbname]

        for dbname in dbnames:
            if self.schema is self.ALL_SCHEMAS:
                schemas = databases[dbname]
            else:
                schemas = [self.schema]
            for schema in schemas:
                yield self.__class__(
                    acl=self.acl,
                    dbname=dbname,
                    schema=schema,
                    role=self.role,
                )


class AclSet(set):
    def expanditems(self, databases):
        for item in self:
            for expansion in item.expand(databases):
                yield expansion
