class Acl(object):
    def __init__(self, name, inspect, grant=None, revoke=None):
        self.name = name
        self.inspect = inspect
        self.grant = grant
        self.revoke = revoke

    def __str__(self):
        return self.name


class AclItem(object):
    def __init__(self, acl, dbname=None, schema=None, role=None):
        self.acl = acl
        self.dbname = dbname
        self.schema = schema
        self.role = role

    def __str__(self):
        return '%(acl)s on %(dbname)s.%(schema)s to %(role)s' % dict(
            self.__dict__,
            schema=self.schema or '__common__'
        )

    def __hash__(self):
        return hash(self.as_tuple())

    def __eq__(self, other):
        return self.as_tuple() == other.as_tuple()

    def as_tuple(self):
        return (self.acl, self.dbname, self.schema, self.role)

    @classmethod
    def from_row(cls, *args):
        return cls(*args)


class AclSet(set):
    pass
