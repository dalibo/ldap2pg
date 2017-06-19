def test_main(mocker):
    environ = dict(
        LDAP_HOST='x',
        LDAP_BIND='x',
        LDAP_BASE='x',
        LDAP_PASSWORD='x',
    )
    mocker.patch('ldap2pg.script.os.environ', environ)
    mocker.patch('ldap2pg.script.logging.basicConfig')
    mocker.patch('ldap2pg.script.create_ldap_connection')
    mocker.patch('ldap2pg.script.create_pg_connection')

    from ldap2pg.script import main

    main()


def test_create_ldap(mocker):
    mocker.patch('ldap2pg.script.ldap3.Connection', autospec=True)
    from ldap2pg.script import create_ldap_connection

    conn = create_ldap_connection(
        host='ldap.company.com',
        bind='cn=admin,dc=company,dc=com', password='keepmesecret',
    )

    assert conn


def test_create_pgconn(mocker):
    mocker.patch('ldap2pg.script.psycopg2.connect', autospec=True)

    from ldap2pg.script import create_pg_connection

    conn = create_pg_connection(dsn="")

    assert conn
