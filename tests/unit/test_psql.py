import pytest


def test_pooler(mocker):
    from ldap2pg.psql import Pooler

    connect = mocker.patch('ldap2pg.psql.connect', autospec=True)
    pool = Pooler("")

    pool.getconn()
    assert connect.called is True

    assert 1 == len(pool)

    connect.reset_mock()
    conn = pool.getconn()
    assert not connect.called

    pool.putconn()
    assert 0 == len(pool)
    assert conn.close.called is True


def test_pooler_context_manager(mocker):
    from ldap2pg.psql import Pooler

    connect = mocker.patch('ldap2pg.psql.connect', autospec=True)

    with Pooler("") as pool:
        pool.getconn()
        assert connect.called is True
        assert 1 == len(pool)

    assert 0 == len(pool)


def test_queries(mocker):
    from ldap2pg.psql import Query, expand_queries, execute_queries, UserError

    queries = [
        Query("Default DB", None, "SELECT 'default-db';"),
        Query("Targetted DB", 'onedb', "SELECT 'targetted-db';"),
        Query("All DB", Query.ALL_DATABASES, "SELECT 'targetted-db';"),
    ]

    pool = mocker.Mock(name='pool')
    conn = pool.getconn.return_value

    count = execute_queries(
        pool, expand_queries(queries, ['postgres', 'template1']),
        timer=None,  # unused when dry
        dry=True,
    )

    assert not conn.execute.called
    assert 4 == count

    count = execute_queries(
        pool, expand_queries(queries, ['postgres', 'template1']),
        mocker.MagicMock(name='timer'),
        dry=False,
    )

    assert conn.execute.called is True

    conn.execute.side_effect = Exception()
    with pytest.raises(UserError):
        execute_queries(
            pool, expand_queries(queries, ['postgres', 'template1']),
            mocker.MagicMock(name='timer'),
            dry=False,
        )


def test_libpq_version(mocker):
    # Remove __libpq_version__ if any.
    mocker.patch('ldap2pg.psql.psycopg2', object())

    from ldap2pg.psql import libpq_version

    assert libpq_version()
