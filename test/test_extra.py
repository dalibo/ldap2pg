# Test order matters.

import pytest


@pytest.fixture(scope='module')
def extrarun(ldap2pg):
    ldap2pg = ldap2pg.bake(c='test/ldap2pg.extra.yml')

    # Ensure database is not sync.
    ldap2pg('--check', _ok_code=1)

    # Synchronize all
    ldap2pg('--real')
    ldap2pg('--check')
    return ldap2pg


def test_roles(extrarun, psql):
    roles = list(psql.roles())
    assert 'charles' in roles


def test_sub_search(extrarun, psql):
    comment = psql.scalar("""\
    SELECT description
    FROM pg_shdescription
    WHERE description LIKE 'cn=solene,%: solene@ldap2pg.docker';
    """)
    assert comment


def test_role_config(extrarun, psql):
    expected = {
        'client_min_messages': 'NOTICE',
        'application_name': 'created',
    }
    assert expected == psql.config('charles')

    expected = {
        'client_min_messages': 'NOTICE',
        'application_name': 'updated',
    }
    assert expected == psql.config('alain')

    assert {} == psql.config('alice')

    expected_unmodified_config = {
        'client_min_messages': 'NOTICE',
        'application_name': 'keep-me',
    }
    assert expected_unmodified_config == psql.config('nicolas')
