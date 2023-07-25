# Test order matters.

from __future__ import unicode_literals

import pytest

from conftest import PSQL


@pytest.mark.go
def test_run(ldap2pg, psql):
    # type: (PSQL) -> None

    c = 'tests/func/ldap2pg.full.yml'

    # Ensure database is not sync.
    ldap2pg('--check', c=c, _ok_code=1)

    # Synchronize all
    ldap2pg('--real', c=c)
    ldap2pg('--check', c=c)

    roles = list(psql.roles())

    assert 'Alan' in roles
    assert 'oscar' not in roles

    assert 'ALICE' in psql.superusers()

    writers = list(psql.members('writers'))

    assert 'daniel' in writers
    assert 'david' in writers
    assert 'didier' in writers
    assert 'ALICE' in psql.members('ldap_roles')

    comment = psql.scalar("""\
    SELECT description
    FROM pg_shdescription
    WHERE description = 'mail: alice@ldap2pg.docker';
    """)
    assert comment


@pytest.mark.go
def test_role_config(ldap2pg, psql):
    # type: (PSQL) -> None

    c = 'tests/func/ldap2pg.config.yml'

    roles = list(psql.roles())

    # Test config_test_update before sync
    assert 'config_test_update' in roles
    update_config = psql.config('config_test_update')
    expected_update_config = {
        'log_statement': 'all',
        'application_name': 'config_test_role_update_old_application_name',
    }
    assert expected_update_config == update_config

    # Test config_test_reset before sync
    assert 'config_test_reset' in roles
    reset_config = psql.config('config_test_reset')
    expected_reset_config = {
        'log_statement': 'none',
        'application_name': 'config_test_role_reset_old_application_name',
    }
    assert expected_reset_config == reset_config

    # Test config_test_unmodified before sync
    assert 'config_test_unmodified' in roles
    unmodified_config = psql.config('config_test_unmodified')
    expected_unmodified_config = {
        'log_statement': 'mod',
        'application_name': 'config_test_role_unmodified_old_application_name',
    }
    assert expected_unmodified_config == unmodified_config

    # Synchronize all
    ldap2pg('--real', c=c)

    roles = list(psql.roles())

    # Test config_test_create after sync
    assert 'config_test_create' in roles
    create_config = psql.config('config_test_create')
    expected_create_config = {
        'log_statement': 'ddl',
        'application_name': 'config_test_create_application_name',
    }
    assert expected_create_config == create_config

    # Test config_test_update after sync
    assert 'config_test_update' in roles
    update_config = psql.config('config_test_update')
    expected_update_config = {
        'log_statement': 'mod',
        'application_name': 'config_test_update_application_name',
    }
    assert expected_update_config == update_config

    # Test config_test_reset after sync
    assert 'config_test_reset' in roles
    reset_config = psql.config('config_test_reset')
    expected_reset_config = {}
    assert expected_reset_config == reset_config

    # Test config_test_unmodified after sync
    assert 'config_test_unmodified' in roles
    unmodified_config = psql.config('config_test_unmodified')
    expected_unmodified_config = {
        'log_statement': 'mod',
        'application_name': 'config_test_role_unmodified_old_application_name',
    }
    assert expected_unmodified_config == unmodified_config
