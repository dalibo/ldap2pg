import pytest


def test_psql(mocker):
    connect = mocker.patch('ldap2pg.psql.psycopg2.connect')

    from ldap2pg.psql import PSQL
    conn = connect.return_value
    cursor = conn.cursor.return_value

    psql = PSQL('')
    session = psql('postgres')
    assert 'postgres' in session.connstring

    with session:
        assert connect.called is True
        assert session.cursor

        sql = session.mogrify('SQL')
        assert sql

        rows = session('SQL')
        assert rows

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
