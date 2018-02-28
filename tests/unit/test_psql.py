import pytest


def test_connstring():
    from ldap2pg.psql import inject_database_in_connstring

    dsns = [
        '',
        'postgres://toto@localhost',
        'postgres://toto@localhost?connect_timeout=4',
        'postgres://toto@localhost/?connect_timeout=4',
        'dbname=other',
        "dbname = 'other'",
        'postgres://toto@localhost/other',
    ]

    for dsn in dsns:
        connstring = inject_database_in_connstring(dsn, 'postgres')
        if dsn.startswith('postgres://'):
            assert 'localhost/postgres' in connstring
        else:
            assert 'dbname=postgres' in connstring

        assert 'other' not in connstring
        assert dsn == inject_database_in_connstring(dsn, None)


def test_psql(mocker):
    connect = mocker.patch('ldap2pg.psql.psycopg2.connect')

    from ldap2pg.psql import PSQL
    conn = connect.return_value
    cursor = conn.cursor.return_value

    psql = PSQL()
    session = psql('postgres')

    with session:
        assert connect.called is True
        assert session.cursor

        sql = session.mogrify('SQL')
        assert sql

        rows = session('SQL')
        assert rows

    connect.reset_mock()
    with session:
        assert connect.called is False

    del psql, session

    assert cursor.close.called is True
    assert conn.close.called is True


def test_psql_pool_limit():
    from ldap2pg.psql import PSQL, UserError

    psql = PSQL(max_pool_size=1)
    # Open one session
    session0 = psql('postgres')

    with pytest.raises(UserError):
        psql('template1')

    session0_bis = psql('postgres')

    assert session0 is session0_bis


def test_iter_sessions(mocker):
    connect = mocker.patch('ldap2pg.psql.psycopg2.connect')

    from ldap2pg.psql import PSQL

    psql = PSQL()

    databases = ['postgres', 'backend', 'frontend']
    for dbname, session in psql.itersessions(databases):
        assert dbname in databases
        assert connect.called is True
        connect.reset_mock()


def test_query():
    from ldap2pg.psql import Query

    qry = Query('Message.', 'postgres', 'SELECT %s;', ('args',))

    assert 2 == len(qry.args)
    assert 'postgres' == qry.dbname
    assert 'Message.' == str(qry)


def test_expand_queries():
    from ldap2pg.psql import Query, expandqueries

    queries = [
        Query('Message.', Query.ALL_DATABASES, 'SELECT 1;'),
        Query('Message.', 'postgres', 'SELECT 1;'),
    ]

    assert '__ALL_DATABASES__' in repr(queries)

    databases = ['postgres', 'template1']
    allqueries = list(expandqueries(queries, databases))

    assert 3 == len(allqueries)


def test_group_by_sessions(mocker):
    PSQLSession = mocker.patch('ldap2pg.psql.PSQLSession', mocker.MagicMock())
    # Identify each with on PSQLSession
    PSQLSession.return_value.__enter__.side_effect = ['a', 'b', 'c']

    from ldap2pg.psql import PSQL, Query

    psql = PSQL()
    queries = [
        Query('M.', None, 'SELECT 1;'),
        Query('M.', None, 'SELECT 2;'),
        Query('M.', 'other', 'SELECT 3;'),
        Query('M.', None, 'SELECT 4;'),
    ]

    sessions = [
        session for session, _ in psql.iter_queries_by_session(queries)]

    assert len(queries) == len(sessions)
    # Ensure the session is reused for the second query
    assert sessions[0] == sessions[1]
    # But not for the last one.
    assert sessions[0] != sessions[3]


def test_run_queries(mocker):
    iqbs = mocker.patch('ldap2pg.psql.PSQL.iter_queries_by_session')
    from ldap2pg.psql import PSQL, Query, UserError

    psql = PSQL()

    queries = [
        Query('q0', None, 'SQL 0'),
        Query('q1', 'postgres', 'SQL 1'),
    ]
    session = mocker.Mock(name='session')
    iqbs.return_value = [(session, query) for query in queries]

    # Dry run
    psql.dry = True
    count = psql.run_queries(queries)
    assert session.called is False
    assert 2 == count

    # Real mode
    session.side_effect = RuntimeError()
    psql.dry = False
    with pytest.raises(UserError):
        psql.run_queries(queries=queries)
    assert session.called is True
