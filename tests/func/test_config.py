import os

import pytest


def test_help():
    from sh import ldap2pg

    ldap2pg('-?')
    ldap2pg('--help')


def test_various_arguments(dev):
    from sh import ldap2pg

    ldap2pg('-vn', '--color', '--config', 'ldap2pg.yml')


YAML_FMT = """\
ldap:
  uri: %(LDAPURI)s
  password: %(LDAPPASSWORD)s

sync_map:
- ldap:
    base: cn=dba,ou=groups,dc=ldap,dc=ldap2pg,dc=docker
    filter: "(objectClass=groupOfNames)"
    attribute: member
  role:
    name_attribute: member.cn
    options: LOGIN SUPERUSER NOBYPASSRLS
"""


def test_custom_yaml():
    from sh import ErrorReturnCode, chmod, ldap2pg, rm

    LDAP2PG_CONFIG = 'my-test-ldap2pg.yml'
    rm('-f', LDAP2PG_CONFIG)
    with pytest.raises(ErrorReturnCode):
        ldap2pg(_env=dict(os.environ, LDAP2PG_CONFIG=LDAP2PG_CONFIG))

    yaml = YAML_FMT % os.environ
    with open(LDAP2PG_CONFIG, 'w') as fo:
        fo.write(yaml)

    # Purge env from value set in file. Other are reads from ldaprc.
    blacklist = ('LDAPURI', 'LDAPHOST', 'LDAPPORT', 'LDAPPASSWORD')
    ldapfree_env = dict(
        (k, v)
        for k, v in os.environ.items()
        if k not in blacklist
    )

    # Ensure world readable password is denied
    with pytest.raises(ErrorReturnCode):
        ldap2pg(config=LDAP2PG_CONFIG, _env=ldapfree_env)

    # And that fixing file mode do the trick.
    chmod('0600', LDAP2PG_CONFIG)
    ldap2pg('--config', LDAP2PG_CONFIG, _env=ldapfree_env)


def test_stdin():
    from sh import ldap2pg

    out = ldap2pg('--config=-', _in="- role: stdinuser")

    assert b'stdinuser' in out.stderr


@pytest.mark.xfail(
    'CI' in os.environ,
    reason="Can't setup SASL on CircleCI")
def test_sasl():
    from sh import ldap2pg

    env = dict(os.environ, LDAPUSER='testsasl', LDAPPASSWORD='voyage')
    out = ldap2pg(config='ldap2pg.yml', verbose=True, _env=env)

    assert b'SASL' in out.stderr
