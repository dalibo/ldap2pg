# coding: utf-8
# Test order matters.

import pytest


@pytest.fixture(scope='module')
def extrarun(ldap2pg):
    ldap2pg = ldap2pg.bake(c='test/extra.ldap2pg.yml')

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
    WHERE description LIKE 'CN=solene,%: solene@bridoulou.fr';
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
    assert expected == psql.config('alter')

    assert {} == psql.config(u'alizée')

    expected_unmodified_config = {
        'client_min_messages': 'NOTICE',
        'application_name': 'keep-me',
    }
    assert expected_unmodified_config == psql.config('nicolas')
