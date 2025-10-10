import os

import pytest


def test_help(ldap2pg):
    ldap2pg('-?')
    ldap2pg('--help')


def test_version(ldap2pg):
    assert "ldap2pg" in ldap2pg("--version")


def ldapfree_env():
    blacklist = ('LDAPURI', 'LDAPHOST', 'LDAPPORT', 'LDAPPASSWORD')
    return dict(
        (k, v)
        for k, v in os.environ.items()
        if k not in blacklist
    )


def test_stdin(ldap2pg, capsys):
    ldap2pg(
        '--config=-',
        _in="version: 6\nrules:\n- role: stdinuser",
        _env=ldapfree_env(),
    )

    _, err = capsys.readouterr()
    assert 'stdinuser' in err


@pytest.mark.xfail(
    'CI' not in os.environ,
    reason="Set CI=true to run GSSAPI test."
)
def test_sasl(ldap2pg, capsys):
    env = dict(
        os.environ,
        LDAPSASL_MECH='GSSAPI',
        LDAPSASL_AUTHCID='Administrator',
    )
    ldap2pg(config='ldap2pg.yml', verbose=True, _env=env)

    _, err = capsys.readouterr()
    assert 'SASL' in err
