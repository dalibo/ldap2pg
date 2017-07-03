def test_psql(mocker):
    connect = mocker.patch('ldap2pg.psql.psycopg2.connect')

    from ldap2pg.psql import PSQL

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

    assert session.cursor is None
