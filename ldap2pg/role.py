from __future__ import unicode_literals

from collections import OrderedDict


class Role(object):
    def __init__(self, name, options=None):
        self.name = name
        self.options = RoleOptions(options or {})

    def __eq__(self, other):
        return self.name == str(other)

    def __hash__(self):
        return hash(self.name)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.name)

    def __str__(self):
        return self.name

    @classmethod
    def from_row(cls, name, *row):
        self = Role(name=name)
        self.options.update_from_row(row)
        return self


class RoleOptions(dict):
    COLUMNS_MAP = OrderedDict([
        ('BYPASSRLS', 'rolbypassrls'),
        ('LOGIN', 'rolcanlogin'),
        ('CREATEDB', 'rolcreatedb'),
        ('CREATEROLE', 'rolcreaterole'),
        ('REPLICATION', 'rolreplication'),
        ('SUPERUSER', 'rolsuper'),
    ])

    def __init__(self, *a, **kw):
        super(RoleOptions, self).__init__(
            BYPASSRLS=False,
            LOGIN=False,
            CREATEDB=False,
            CREATEROLE=False,
            REPLICATION=False,
            SUPERUSER=False,
        )
        init = dict(*a, **kw)
        self.update(init)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self)

    def __str__(self):
        return ' '.join((
            ('NO' if value is False else '') + name
            for name, value in self.items()
        ))

    def update_from_row(self, row):
        self.update(dict(zip(self.COLUMNS_MAP.keys(), row)))

    def update(self, other):
        spurious_options = set(other.keys()) - set(self.keys())
        if spurious_options:
            message = "Unknown options %s" % (', '.join(spurious_options),)
            raise ValueError(message)
        return super(RoleOptions, self).update(other)


class RoleSet(set):
    def __init__(self, *a, **kw):
        super(RoleSet, self).__init__(*a, **kw)

    def reindex(self):
        return {
            role.name: role
            for role in self
        }

    def diff(self, other):
        # Yields SQL queries to synchronize self with other.
        spurious = self - other
        for role in spurious:
            yield 'DROP ROLE %s;' % (role.name)

        existing = self & other
        myindex = self.reindex()
        itsindex = other.reindex()
        for role in existing:
            my = myindex[role.name]
            its = itsindex[role.name]
            if my.options == its.options:
                continue
            yield 'ALTER ROLE %s WITH %s;' % (role.name, its.options)

        missing = other - self
        for role in missing:
            yield 'CREATE ROLE %s WITH %s;' % (role.name, role.options)
