# Test order matters.


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('-v')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'spurious' in roles
    assert 'alice' in superusers


def test_real_mode(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('-vN')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'bob' in roles
    assert 'spurious' not in roles
    assert 'alice' in superusers

    assert 'foo' in psql.members('app0')
    assert 'bar' in psql.members('app1')
    assert 'alice' in psql.members('ldap_users')
