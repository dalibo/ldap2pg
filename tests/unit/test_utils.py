import pytest


def test_deep_getset():
    from ldap2pg.utils import deepget, deepset

    a = dict()

    deepset(a, 'toto:tata', 1)

    assert 1 == a['toto']['tata']
    assert 1 == deepget(a, 'toto:tata')

    with pytest.raises(KeyError):
        deepget(a, 'toto:titi')
