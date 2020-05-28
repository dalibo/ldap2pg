import logging
import os
import sys
from functools import partial

import pytest
import sh


class PSQL(object):
    # A helper object to do SQL queries with real psql.
    def __init__(self):
        from sh import psql
        self.psql = psql

    def __call__(self, *a, **kw):
        return self.psql(*a, **kw)

    def scalar(self, select, *a, **kw):
        return next(self.select1(select, *a, **kw))

    def select1(self, select, *a, **kw):
        # Execute a SELECT and yield each line as a single value.
        return filter(None, (
            line.strip()
            for line in self('-tc', select, *a, _iter=True, **kw)
        ))

    def members(self, role):
        # List members of role
        return self.select1(
            # Good old SQL injection. Who cares?
            "SELECT m.rolname FROM pg_roles AS m "
            "JOIN pg_auth_members a ON a.member = m.oid "
            "JOIN pg_roles AS r ON r.oid = a.roleid "
            " WHERE r.rolname = '%s' "
            "ORDER BY 1;" % (role,)
        )

    def roles(self):
        # List **all** roles
        return self.select1("SELECT rolname FROM pg_roles;")

    def superusers(self):
        # List superusers
        return self.select1(
            "SELECT rolname FROM pg_roles WHERE rolsuper IS TRUE;"
        )

    def tables(self, *a, **kw):
        # List tables
        return self.select1(
            "SELECT relname "
            "FROM pg_catalog.pg_class c "
            "JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace "
            "WHERE "
            "    c.relkind = 'r' "
            "    AND n.nspname !~ '^pg_' "
            "    AND n.nspname <> 'information_schema' "
            "ORDER BY 1;",
            *a, **kw
        )


@pytest.fixture(scope='session')
def psql():
    # Supply the PSQL helper as a pytest fixture.
    return PSQL()


class LDAP(object):
    # Helper to query LDAP with creds from envvars.
    def __init__(self):
        self.common_args = (
            '-xv',
            '-w', os.environ['LDAPPASSWORD'],
        )

        self.search = sh.ldapsearch.bake(*self.common_args)

    def search_sub_dn(self, base):
        # Iter dn under base entry, excluded.
        for line in self.search('-b', base, 'dn', _iter=True):
            if not line.startswith('dn: '):
                continue

            if line.startswith('dn: ' + base):
                continue

            yield line.strip()[len('dn: '):]


@pytest.fixture(scope='session')
def ldap():
    # Supply LDAP helper as a pytest fixture
    #
    # def test_rockon(ldap):
    #     entries = ldap.search(...)
    return LDAP()


@pytest.fixture(scope='module', autouse=True)
def resetpostgres():
    from sh import Command

    Command('fixtures/postgres.sh')()


def lazy_write(attr, data):
    # Lazy access sys.{stderr,stdout} to mix with capsys.
    getattr(sys, attr).write(data)
    return False  # should_quit


@pytest.fixture(scope='session', autouse=True)
def sh_errout():
    logging.getLogger('sh').setLevel(logging.ERROR)

    # Duplicate tested command stdio to pytest capsys.
    sh._SelfWrapper__self_module.Command._call_args.update(dict(
        err=partial(lazy_write, 'stderr'),
        out=partial(lazy_write, 'stdout'),
        tee=True,
    ))
