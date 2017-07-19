import pytest


def test_psql(mocker):
    connect = mocker.patch('ldap2pg.psql.psycopg2.connect')

    from ldap2pg.psql import PSQL
    conn = connect.return_value
    cursor = conn.cursor.return_value

    dsns = [
        '',
        'postgres://toto@localhost',
        'postgres://toto@localhost?connect_timeout=4',
        'postgres://toto@localhost/?connect_timeout=4',
    ]
    for dsn in dsns:
        psql = PSQL(dsn)
        session = psql('postgres')
        if dsn.startswith('postgres://'):
            assert 'localhost/postgres' in session.connstring
        else:
            assert 'dbname=postgres' in session.connstring

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
