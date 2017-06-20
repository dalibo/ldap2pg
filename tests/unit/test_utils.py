import pytest


def test_deep_update():
    from ldap2pg.utils import deepupdate

    a = dict(toto=dict(tata=1))
    b = dict(toto=dict(titi=2))

    deepupdate(a, b)

    assert dict(tata=1, titi=2) == a['toto']


def test_deep_getset():
    from ldap2pg.utils import deepget, deepset

    a = dict()

    deepset(a, 'toto:tata', 1)

    assert 1 == a['toto']['tata']
    assert 1 == deepget(a, 'toto:tata')

    with pytest.raises(KeyError):
        deepget(a, 'toto:titi')
