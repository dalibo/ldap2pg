from .psql import Query
from .utils import AllDatabases


class Acl(object):
    def __init__(self, name, inspect=None, grant=None, revoke=None):
        self.name = name
        self.inspect = inspect
        self.grant_sql = grant
        self.revoke_sql = revoke

    def __lt__(self, other):
        return str(self) < str(other)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __str__(self):
        return self.name

    def grant(self, item):
        return Query(
            "Grant %s." % (item,),
            item.dbname,
            self.grant_sql % dict(
                database=item.dbname,
                schema=item.schema,
                role=item.role,
            ),
        )

    def revoke(self, item):
        return Query(
            "Revoke %s." % (item,),
            item.dbname,
            self.revoke_sql % dict(
                database=item.dbname,
                schema=item.schema,
                role=item.role,
            ),
        )


class AclItem(object):
    ALL_DATABASES = AllDatabases()

    @classmethod
    def from_row(cls, *args):
        return cls(*args)

    def __init__(self, acl, dbname=None, schema=None, role=None):
        self.acl = acl
        self.dbname = dbname
        self.schema = schema
        self.role = role

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

    def expand(self, databases):
        if self.dbname is self.ALL_DATABASES:
            for dbname in databases:
                yield self.__class__(
                    acl=self.acl,
                    dbname=dbname,
                    schema=self.schema,
                    role=self.role,
                )
        else:
            yield self


class AclSet(set):
    def expanditems(self, databases):
        for item in self:
            for expansion in item.expand(databases):
                yield expansion
