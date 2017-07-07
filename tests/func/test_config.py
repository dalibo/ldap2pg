import os

import pytest


def test_help():
    from sh import ldap2pg

    ldap2pg('-?')
    ldap2pg('--help')


def test_various_arguments():
    from sh import ldap2pg

    ldap2pg('-vn', '--color')


def test_versionned_yaml(dev):
    from sh import ldap2pg

    ldap2pg(config='ldap2pg.yml')
    ldap2pg(config='ldap2pg.master.yml')


YAML_FMT = """\
ldap:
  host: %(LDAP_HOST)s
  port: %(LDAP_PORT)s
  bind: %(LDAP_BIND)s
  password: %(LDAP_PASSWORD)s

sync_map:
- ldap:
    base: cn=dba,ou=groups,dc=ldap2pg,dc=local
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

    yaml = YAML_FMT % dict(LDAP_PORT=389, **os.environ)
    with open(LDAP2PG_CONFIG, 'w') as fo:
        fo.write(yaml)

    ldapfree_env = {
        k: v
        for k, v in os.environ.items()
        if not k.startswith('LDAP_')
    }

    # Ensure world readable password is denied
    with pytest.raises(ErrorReturnCode):
        ldap2pg(_env=ldapfree_env)

    # And that fixing file mode do the trick.
    chmod('0600', LDAP2PG_CONFIG)
    ldap2pg('--config', LDAP2PG_CONFIG, _env=ldapfree_env)


def test_stdin():
    from sh import ldap2pg

    out = ldap2pg('--config=-', _in="- role: stdinuser")

    assert b'stdinuser' in out.stderr
