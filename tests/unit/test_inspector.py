# -*- coding: utf-8 -*-

from __future__ import unicode_literals

import pytest


def test_generic_fetch(mocker):
    from ldap2pg.inspector import psycopg2, PostgresInspector, UserError

    inspector = PostgresInspector(
        pool=mocker.Mock(name='pool'),
        raw_sql='SELECT 1;',
        flat_list=['val0'],
        tuple_list=[['val0']],
    )
    inspector.pool.getconn.side_effect = psycopg2.ProgrammingError()

    with pytest.raises(UserError):
        inspector.fetch('raw_sql')

    assert [] == inspector.fetch(None)
    assert [('val0',)] == inspector.fetch('flat_list')
    assert [['val0']] == inspector.fetch('tuple_list')


def test_format_roles_inspect_sql(mocker):
    from ldap2pg.inspector import PostgresInspector

    inspector = PostgresInspector(
        all_roles='SELECT {options}',
        custom_null=None,
        custom_list=['user'],
    )

    assert 'rolsuper' in inspector.format_roles_query()
    assert inspector.format_roles_query(name='custom_null') is None
    assert ['user'] == inspector.format_roles_query(name='custom_list')


def test_filter_roles():
    from ldap2pg.inspector import PostgresInspector, Role

    inspector = PostgresInspector()
    inspector.roles_blacklist = ['pg_*', 'postgres']

    allroles = [
        Role('postgres'),
        Role('pg_signal_backend'),
        Role('dba', members=['alice']),
        Role('alice'),
        Role('unmanaged'),
    ]
    managedroles = {'alice', 'dba', 'public'}
    allroles, managedroles = inspector.filter_roles(
        allroles, managedroles)

    assert 3 == len(allroles)
    assert 3 == len(managedroles)
    assert 'dba' in allroles
    assert 'alice' in allroles
    assert 'unmanaged' in allroles
    assert 'unmanaged' not in managedroles
    assert 'postgres' not in allroles
    assert 'postgres' not in managedroles
    assert 'public' in managedroles


def test_process_grants(mocker):
    from ldap2pg.inspector import PostgresInspector, UserError

    inspector = PostgresInspector()
    priv = mocker.Mock(grant_sql='IN {schema} TO {role}')
    priv.name = 'connect'
    rows = [
        (None, 'postgres', True),
        (None, 'pg_signal_backend'),  # Old signature, fallback to True
        ('public', 'alice', True),
    ]

    items = sorted(inspector.process_grants(priv, 'postgres', rows))

    assert 3 == len(items)
    item = items[0]
    assert 'connect' == item.privilege
    assert 'postgres' == item.dbname
    assert 'public' == item.schema
    assert 'alice' == item.role

    # Schema naïve privilege
    priv.grant_sql = 'TO {role}'
    rows = [('public', 'alice')]
    items = sorted(inspector.process_grants(priv, 'postgres', rows))
    assert items[0].schema is None

    with pytest.raises(UserError):
        list(inspector.process_grants(priv, 'db', [('incomplete',)]))


def test_schemas_global_owners(mocker):
    from ldap2pg.inspector import Database, PostgresInspector

    inspector = PostgresInspector(
        schemas=['public'],
        owners=['owner', 'postgres'],
    )
    inspector.roles_blacklist = ['postgres']

    db = Database('db', owner='postgres')
    inspector.fetch_schemas(databases=[db])

    assert 'public' in db.schemas
    assert 'owner' in db.schemas['public'].owners
    assert 'postgres' not in db.schemas['public'].owners


def test_schemas_with_owners(mocker):
    from ldap2pg.inspector import Database, PostgresInspector

    inspector = PostgresInspector(
        schemas=[
            ('public', ['pubowner', 'postgres']),
            ('ns', ['nsowner']),
        ],
        owners=['owner']
    )

    db = Database('db', owner='postgres')
    inspector.fetch_schemas(
        databases=[db], managedroles={'pubowner', 'nsowner'})

    assert 'public' in db.schemas
    assert 'pubowner' in db.schemas['public'].owners
    assert 'owner' not in db.schemas['public'].owners
    assert 'postgres' not in db.schemas['public'].owners
    assert 'ns' in db.schemas
    assert 'nsowner' in db.schemas['ns'].owners


def test_grants(mocker):
    pg = mocker.patch(
        'ldap2pg.inspector.PostgresInspector.process_grants', autospec=True)

    from ldap2pg.inspector import Database, Grant, PostgresInspector, Schema
    from ldap2pg.privilege import NspAcl

    privileges = dict(
        noinspect=NspAcl(name='noinspect'),
        ro=NspAcl(name='ro', inspect='SQL'),
    )

    pg.return_value = [
        Grant('ro', 'db', None, 'alice'),
        Grant('ro', 'db', None, 'public'),
        Grant('ro', 'db', None, 'unmanaged'),
        Grant('ro', 'db', 'unmanaged', 'alice'),
        Grant('ro', 'db', None, 'alice', owner='unmanaged'),
    ]

    pool = mocker.Mock(name='pool')
    conn = pool.getconn.return_value
    conn.query.return_value = []
    inspector = PostgresInspector(pool=pool, privileges=privileges)

    db = Database('db', owner='postgres')
    db.schemas['public'] = Schema('public', owners=['owner'])
    grants = inspector.fetch_grants(
        databases=[db],
        roles=['alice', 'public'],
    )

    assert 2 == len(grants)
    grantees = [a.role for a in grants]
    assert 'public' in grantees
    assert 'alice' in grantees


def test_grants_cached(mocker):
    cls = 'ldap2pg.inspector.PostgresInspector'
    pg = mocker.patch(cls + '.process_grants', autospec=True)
    mocker.patch(cls + '.fetch_shared_query', autospec=True)

    from ldap2pg.inspector import Database, PostgresInspector, Schema
    from ldap2pg.privilege import NspAcl

    privileges = dict(
        cached=NspAcl(
            'cached', inspect=dict(shared_query='shared', keys=['CACHED']))
    )

    pg.return_value = []
    pool = mocker.Mock(name='pool')
    conn = pool.getconn.return_value
    conn.query.return_value = []
    inspector = PostgresInspector(pool=pool, privileges=privileges)

    db = Database('db', owner='postgres')
    db.schemas['public'] = Schema('public', owners=['owner'])
    grants = inspector.fetch_grants(
        databases=[db],
        roles=['alice', 'public'],
    )

    assert 0 == len(grants)


def test_fetch_cached_query(mocker):
    from ldap2pg.inspector import PostgresInspector

    shared_queries = dict(shared="SELECT pouet;")
    inspector = PostgresInspector(shared_queries=shared_queries)

    conn = mocker.Mock(name='conn')
    conn.query.return_value = [
        ('KEY0', 'public', 'alice'),
        ('KEY0', 'public', 'alain'),
        ('KEY1', 'public', 'alice'),
        ('KEY2', 'public', 'adrien'),
        ('KEY2', 'public', 'armand'),
    ]

    rows = inspector.fetch_shared_query('shared', ['KEY0'], 'db0', conn)
    assert 2 == len(rows)

    conn.reset_mock()
    rows = inspector.fetch_shared_query(
        'shared', ['KEY1', 'KEY2'], 'db0', conn)
    assert 3 == len(rows)
    assert not conn.called


def test_databases(mocker):
    from ldap2pg.inspector import Database, PostgresInspector

    inspector = PostgresInspector(
        pool=mocker.MagicMock(name='pool'),
        databases=['postgres'],
    )

    conn = inspector.pool.getconn.return_value
    conn.query.return_value = [Database('postgres', 'owner')]

    databases = inspector.fetch_databases()

    assert 'postgres' in databases


def test_me(mocker):
    from ldap2pg.inspector import PostgresInspector

    inspector = PostgresInspector(
        pool=mocker.MagicMock(name='pool'),
    )
    conn = inspector.pool.getconn.return_value
    conn.queryone.return_value = ('postgres', True)
    name, issuper = inspector.fetch_me()

    assert 'postgres' == name
    assert issuper


def test_roles(mocker):
    from ldap2pg.inspector import PostgresInspector

    inspector = PostgresInspector(
        psql=mocker.MagicMock(name='psql'),
        all_roles=['precreated', 'spurious'],
        roles_blacklist_query=['postgres'],
        managed_roles=None,
    )

    pgallroles, pgmanagedroles = inspector.fetch_roles()

    assert 'precreated' in pgallroles
    assert 'spurious' in pgallroles
    assert pgallroles < pgmanagedroles

    inspector.queries['managed_roles'] = ['precreated']

    _, pgmanagedroles = inspector.fetch_roles()

    assert 'spurious' not in pgmanagedroles

    blacklist = inspector.fetch_roles_blacklist()
    assert ['postgres'] == blacklist
