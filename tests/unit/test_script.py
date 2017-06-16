def test_main(mocker):
    environ = dict(
        LDAP_HOST='x',
        LDAP_BIND='x',
        LDAP_BASE='x',
        LDAP_PASSWORD='x',
    )
    mocker.patch('ldap2pg.script.os.environ', environ)
    mocker.patch('ldap2pg.script.logging.basicConfig')
    mocker.patch('ldap2pg.script.psycopg2.connect')
    mocker.patch('ldap2pg.script.ldap3.Connection')

    from ldap2pg.script import main

    main()
