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
        _in="version: 5\nsync_map:\n- role: stdinuser",
        _env=ldapfree_env(),
    )

    _, err = capsys.readouterr()
    assert 'stdinuser' in err


@pytest.mark.xfail(
    'CI' in os.environ,
    reason="Can't setup SASL on CircleCI")
def test_sasl(ldap2pg, capsys):
    env = dict(
        os.environ,
        # py-ldap2pg reads non-standard var USER.
        LDAPUSER='testsasl',
        # ldap2pg requires explicit SASL_MECH, and standard SASL_AUTHID.
        LDAPSASL_MECH='DIGEST-MD5',
        LDAPSASL_AUTHCID='testsasl',
        LDAPPASSWORD='voyage',
    )
    ldap2pg(config='ldap2pg.yml', verbose=True, _env=env)

    _, err = capsys.readouterr()
    assert 'SASL' in err
