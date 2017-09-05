# Test order matters.

from __future__ import unicode_literals

import pytest


def test_dry_run(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--verbose', '--config', 'tests/func/ldap2pg.yml')
    roles = list(psql.roles())
    superusers = list(psql.superusers())
    assert 'oscar' in roles
    assert 'alice' in superusers


def test_check_mode(dev, ldap, psql):
    from sh import ldap2pg

    ldap2pg('--check', '--config', 'tests/func/ldap2pg.yml', _ok_code=1)


def test_real_mode(dev, ldap, psql):
    from sh import ErrorReturnCode, ldap2pg

    assert 'keepme' in psql.tables(dbname='legacy')
    # Assert daniel can connect to backend, not to frontend
    psql(U='daniel', d='backend', c='SELECT CURRENT_USER')
    with pytest.raises(ErrorReturnCode):
        psql(U='daniel', d='frontend', c='SELECT CURRENT_USER')

    ldap2pg('-vN', c='tests/func/ldap2pg.yml')

    roles = list(psql.roles())
    frontend = list(psql.members('frontend'))

    assert 'alan' in roles
    assert 'oscar' not in roles

    assert 'alice' in psql.superusers()

    assert 'dave' in psql.members('backend')
    assert 'david' in frontend
    assert 'alice' in psql.members('ldap_users')

    # Assert that table keepme owned by deleted user spurious is not dropped!
    assert 'keepme' in psql.tables(dbname='legacy')

    # Assert CONNECT to backend has been revoked from daniel.
    with pytest.raises(ErrorReturnCode):
        psql(U='daniel', d='backend', c='SELECT CURRENT_USER')
    # Assert daniel can now connect to frontend
    psql(U='daniel', d='frontend', c='SELECT CURRENT_USER')

    # Assert carole can't connect even if she is in groups. This check
    # role_match pattern.
    assert 'carole' in frontend
    with pytest.raises(ErrorReturnCode):
        psql(U='carole', d='frontend', c='SELECT CURRENT_USER')


def test_nothing_to_do():
    from sh import ldap2pg

    out = ldap2pg('--real', '--config', 'tests/func/ldap2pg.yml')

    assert b'Nothing to do' in out.stderr


YAML = """\
postgres:
  roles_query: |
    SELECT
        role.rolname, array_agg(members.rolname) AS members,
        {options}
    FROM
        pg_catalog.pg_roles AS role
    LEFT JOIN pg_catalog.pg_auth_members ON roleid = role.oid
    LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = member
    WHERE role.rolname LIKE 'd%'
    GROUP BY role.rolname, {options}
    ORDER BY 1;

sync_map:
  ldap:
      base: ou=people,dc=ldap,dc=ldap2pg,dc=docker
      filter: "(cn=d*)"
      attribute: cn
  role:
    name_attribute: cn
    options: LOGIN
"""


def test_custom_query(psql):
    from sh import ldap2pg

    # Ensure we have a role not matching `d%`
    roles = list(psql.roles())
    assert 'alan' in roles

    out = ldap2pg('-v', '--config=-', _in=YAML)

    # However, alan is not dopped.
    assert b'Nothing to do' in out.stderr
