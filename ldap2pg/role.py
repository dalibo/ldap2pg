from __future__ import unicode_literals

from collections import OrderedDict
import logging

from .psql import Query
from .utils import unicode


logger = logging.getLogger(__name__)


class Role(object):
    def __init__(self, name, options=None, members=None, parents=None):
        self.name = name
        self.members = members or []
        self.options = RoleOptions(options or {})
        self.parents = parents or []

    def __eq__(self, other):
        return self.name == unicode(other)

    def __hash__(self):
        return hash(self.name)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.name)

    def __str__(self):
        return self.name

    def __lt__(self, other):
        return unicode(self) < unicode(other)

    @classmethod
    def from_row(cls, name, members, *row):
        self = Role(name=name, members=list(filter(None, members)))
        self.options.update_from_row(row)
        return self

    def create(self):
        yield Query(
            'Create %s.' % (self.name,),
            'postgres',
            'CREATE ROLE %s WITH %s;' % (self.name, self.options)
        )
        if self.members:
            yield Query(
                'Add %s members.' % (self.name,),
                'postgres',
                "GRANT %(role)s TO %(members)s;" % dict(
                    members=", ".join(self.members),
                    role=self.name,
                ),
            )

    def alter(self, other):
        # Yields SQL queries to reach other state.

        if self.options != other.options:
            yield Query(
                'Update options of %s.' % (self.name,),
                'postgres',
                'ALTER ROLE %s WITH %s;' % (self.name, other.options),
            )

        if self.members != other.members:
            missing = set(other.members) - set(self.members)
            if missing:
                logger.debug(
                    "Role %s miss members %s.",
                    self.name, ', '.join(missing)
                )
                yield Query(
                    'Add missing %s members.' % (self.name,),
                    'postgres',
                    "GRANT %(role)s TO %(members)s;" % dict(
                        members=", ".join(missing),
                        role=self.name,
                    ),
                )
            spurious = set(self.members) - set(other.members)
            if spurious:
                logger.debug(
                    "Role %s has spurious members %s.",
                    self.name, ', '.join(spurious)
                )
                yield Query(
                    'Delete spurious %s members.' % (self.name,),
                    'postgres',
                    "REVOKE %(role)s FROM %(members)s;" % dict(
                        members=", ".join(spurious),
                        role=self.name,
                    ),
                )

    _drop_objects_sql = """
    DO $$BEGIN EXECUTE 'REASSIGN OWNED BY %(role)s TO ' || session_user; END$$;
    DROP OWNED BY %(role)s;
    """.strip().replace(4 * ' ', '')

    def drop(self):
        yield Query(
            'Reassign %s objects and purge ACL on %%(dbname)s.' % (self.name,),
            Query.ALL_DATABASES,
            self._drop_objects_sql % dict(role=self.name),
        )
        yield Query(
            'Drop %s.' % (self.name,),
            'postgres',
            "DROP ROLE %(role)s;" % dict(role=self.name),
        )


class RoleOptions(dict):
    COLUMNS_MAP = OrderedDict([
        ('BYPASSRLS', 'rolbypassrls'),
        ('LOGIN', 'rolcanlogin'),
        ('CREATEDB', 'rolcreatedb'),
        ('CREATEROLE', 'rolcreaterole'),
        ('INHERIT', 'rolinherit'),
        ('REPLICATION', 'rolreplication'),
        ('SUPERUSER', 'rolsuper'),
    ])

    def __init__(self, *a, **kw):
        super(RoleOptions, self).__init__(
            BYPASSRLS=False,
            LOGIN=False,
            CREATEDB=False,
            CREATEROLE=False,
            INHERIT=True,
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

    def resolve_membership(self):
        index_ = self.reindex()
        for role in self:
            while role.parents:
                parent_name = role.parents.pop()
                parent = index_[parent_name]
                if role.name in parent.members:
                    continue
                logger.debug("Add %s as member of %s.", role.name, parent.name)
                parent.members.append(role.name)

    def reindex(self):
        return {role.name: role for role in self}

    def flatten(self):
        # Generates the flatten tree of roles, children first.

        index = self.reindex()
        seen = set()

        def walk(name):
            if name in seen:
                return
            try:
                role = index[name]
            except KeyError:
                # We are trying to walk a member out of set. This is the case
                # where a role is missing but not one of its member.
                return

            for member in role.members:
                for i in walk(member):
                    yield i
            yield name
            seen.add(name)

        for name in sorted(index.keys()):
            for i in walk(name):
                yield index[i]
