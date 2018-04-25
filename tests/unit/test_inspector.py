from __future__ import unicode_literals

import pytest


def test_generic_fetch(mocker):
    from ldap2pg.inspector import psycopg2, PostgresInspector, UserError

    inspector = PostgresInspector(
        raw_sql='SELECT 1;',
        flat_list=['val0'],
        tuple_list=[['val0']],
    )
    psql = mocker.Mock(name='psql', side_effect=psycopg2.ProgrammingError())

    with pytest.raises(UserError):
        inspector.fetch(psql, 'raw_sql')

    psql = mocker.Mock(name='psql', return_value=[('val0',), ('val1',)])
    rows = inspector.fetch(psql, 'POUET;', inspector.row1)
    assert ['val0', 'val1'] == rows

    assert [] == inspector.fetch(None, None)
    assert [('val0',)] == inspector.fetch(None, 'flat_list')
    assert [['val0']] == inspector.fetch(None, 'tuple_list')


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

    inspector = PostgresInspector(
        roles_blacklist=['pg_*', 'postgres'],
    )

    allroles = [
        Role('postgres'),
        Role('pg_signal_backend'),
        Role('dba', members=['alice']),
        Role('alice'),
        Role('unmanaged'),
    ]
    managedroles = {'alice', 'dba'}
    allroles, managedroles = inspector.filter_roles(
        allroles, managedroles)

    assert 3 == len(allroles)
    assert 2 == len(managedroles)
    assert 'dba' in allroles
    assert 'alice' in allroles
    assert 'unmanaged' in allroles
    assert 'unmanaged' not in managedroles
    assert 'postgres' not in allroles
    assert 'postgres' not in managedroles


def test_process_grants():
    from ldap2pg.inspector import PostgresInspector, UserError

    inspector = PostgresInspector()
    rows = [
        (None, 'postgres', True),
        (None, 'pg_signal_backend'),  # Old signature, fallback to True
        ('public', 'alice', True),
    ]

    items = sorted(inspector.process_grants('connect', 'postgres', rows))

    assert 3 == len(items)
    item = items[0]
    assert 'connect' == item.privilege
    assert 'postgres' == item.dbname
    assert 'public' == item.schema
    assert 'alice' == item.role

    with pytest.raises(UserError):
        list(inspector.process_grants('priv', 'db', [('incomplete',)]))


def test_process_schema_rows():
    from ldap2pg.inspector import PostgresInspector

    inspector = PostgresInspector()

    rows = ['legacy']
    my = dict(inspector.process_schemas(rows))
    assert 'legacy' in my
    assert my['legacy'] is False

    rows = [['public', ['owner']]]
    my = dict(inspector.process_schemas(rows))
    assert 'public' in my
    assert 'owner' in my['public']


def test_schemas_global_owners(mocker):
    from ldap2pg.inspector import PostgresInspector

    psql = mocker.MagicMock()
    psql.itersessions.return_value = [('db', psql)]
    inspector = PostgresInspector(
        psql=psql,
        roles_blacklist=['postgres'],
        schemas=['public'],
        owners=['owner', 'postgres']
    )

    schemas = inspector.fetch_schemas(databases=['db'])

    assert 'db' in schemas
    assert 'public' in schemas['db']
    assert 'owner' in schemas['db']['public']
    assert 'postgres' not in schemas['db']['public']


def test_schemas_with_owners(mocker):
    from ldap2pg.inspector import PostgresInspector

    psql = mocker.MagicMock()
    psql.itersessions.return_value = [('db', psql)]
    inspector = PostgresInspector(
        psql=psql,
        schemas=[
            ('public', ['pubowner', 'postgres']),
            ('ns', ['nsowner']),
        ],
        owners=['owner']
    )

    schemas = inspector.fetch_schemas(
        databases=['db'], managedroles={'pubowner', 'nsowner'})

    assert 'db' in schemas
    assert 'public' in schemas['db']
    assert 'pubowner' in schemas['db']['public']
    assert 'owner' not in schemas['db']['public']
    assert 'postgres' not in schemas['db']['public']
    assert 'ns' in schemas['db']
    assert 'nsowner' in schemas['db']['ns']


def test_grants(mocker):
    pg = mocker.patch(
        'ldap2pg.inspector.PostgresInspector.process_grants', autospec=True)

    from ldap2pg.inspector import PostgresInspector, Grant
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

    psql = mocker.MagicMock(name='psql')
    psql.itersessions.return_value = [('db', psql)]
    inspector = PostgresInspector(psql=psql, privileges=privileges)

    grants = inspector.fetch_grants(
        schemas=dict(db=dict(public=['owner'])),
        roles=['alice'])

    assert 2 == len(grants)
    grantees = [a.role for a in grants]
    assert 'public' in grantees
    assert 'alice' in grantees


def test_roles(mocker):
    from ldap2pg.inspector import PostgresInspector

    inspector = PostgresInspector(
        psql=mocker.MagicMock(name='psql'),
        databases=['postgres'],
        all_roles=['precreated', 'spurious'],
        managed_roles=None,
    )

    databases, pgallroles, pgmanagedroles = inspector.fetch_roles()

    assert 'postgres' in databases
    assert 'precreated' in pgallroles
    assert 'spurious' in pgallroles
    assert pgallroles == pgmanagedroles

    inspector.queries['managed_roles'] = ['precreated']

    _, _, pgmanagedroles = inspector.fetch_roles()

    assert 'spurious' not in pgmanagedroles
