from __future__ import unicode_literals

from collections import OrderedDict
import logging

from .utils import Query


logger = logging.getLogger(__name__)


class Role(object):
    def __init__(self, name, options=None, members=None, parents=None):
        self.name = name
        self.members = members or []
        self.options = RoleOptions(options or {})
        self.parents = parents or []

    def __eq__(self, other):
        return self.name == str(other)

    def __hash__(self):
        return hash(self.name)

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.name)

    def __str__(self):
        return self.name

    def __lt__(self, other):
        return str(self) < str(other)

    @classmethod
    def from_row(cls, name, members, *row):
        self = Role(name=name, members=list(filter(None, members)))
        self.options.update_from_row(row)
        return self

    _members_insert = """
    INSERT INTO pg_catalog.pg_auth_members
    SELECT
        r.oid AS roleid,
        m.oid AS member,
        g.oid AS grantor,
        FALSE AS admin_option
    FROM pg_roles AS r
    JOIN pg_roles AS m ON m.rolname = ANY(%s)
    JOIN pg_roles g ON g.rolname = session_user
    WHERE r.rolname = %s;
    """.replace('\n    ', '\n').strip()

    _members_delete = """
    WITH spurious AS (
        SELECT r.oid AS roleid, m.oid AS member
        FROM pg_roles AS r
        JOIN pg_roles AS m ON m.rolname = ANY(%s)
        WHERE r.rolname = %s
    )
    DELETE FROM pg_catalog.pg_auth_members AS a
    USING spurious
    WHERE a.roleid = spurious.roleid AND a.member = spurious.member;
    """.replace('\n    ', '\n').strip()

    _members_delete_all = """
    WITH roleids AS (
        SELECT r.oid AS roleid
        FROM pg_roles AS r
        WHERE r.rolname = %s
    )
    DELETE FROM pg_catalog.pg_auth_members AS a
    USING roleids
    WHERE a.roleid = roleids.roleid;
    """.replace('\n    ', '\n').strip()

    def create(self):
        yield Query(
            'create %s.' % (self.name,),
            -1,  # rowcount
            'CREATE ROLE %s WITH %s;' % (self.name, self.options)
        )
        if self.members:
            yield Query(
                'add %s members.' % (self.name,),
                len(self.members),  # rowcount
                self._members_insert,
                (self.members, self.name,)
            )

    def alter(self, other):
        # Yields SQL queries to reach other state.

        if self.options != other.options:
            yield Query(
                'update options of %s.' % (self.name,),
                -1,  # rowcount
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
                    'add missing %s members.' % (self.name,),
                    len(missing),  # rowcount
                    self._members_insert,
                    (list(missing), self.name,)
                )
            spurious = set(self.members) - set(other.members)
            if spurious:
                logger.debug(
                    "Role %s has spurious members %s.",
                    self.name, ', '.join(spurious)
                )
                yield Query(
                    'delete spurious %s members.' % (self.name,),
                    len(spurious),  # rowcount
                    self._members_delete,
                    (list(spurious), self.name,)
                )

    def drop(self):
        if self.members:
            yield Query(
                'remove members from %s.' % (self.name,),
                len(self.members),  # rowcount
                self._members_delete_all,
                (self.name,),
            )
        yield Query(
            'drop %s.' % (self.name,),
            -1,  # rowcount
            'DROP ROLE %s;' % (self.name),
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
            for parent_name in role.parents:
                parent = index_[parent_name]
                if role.name in parent.members:
                    continue
                logger.debug("Add %s as member of %s.", role.name, parent.name)
                parent.members.append(role.name)

    def reindex(self):
        return {
            role.name: role
            for role in self
        }

    def diff(self, other):
        # Yields SQL queries to synchronize self with other.

        missing = RoleSet(other - self)
        for role in missing.flatten():
            for qry in role.create():
                yield qry

        existing = self & other
        myindex = self.reindex()
        itsindex = other.reindex()
        for role in existing:
            my = myindex[role.name]
            its = itsindex[role.name]
            for qry in my.alter(its):
                yield qry

        spurious = RoleSet(self - other)
        for role in reversed(list(spurious.flatten())):
            for qry in role.drop():
                yield qry

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
