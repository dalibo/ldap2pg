# coding: utf-8

from __future__ import unicode_literals

import pytest


def test_deep_getset():
    from ldap2pg.utils import deepget, deepset

    a = dict()

    deepset(a, 'toto:tata', 1)

    assert 1 == a['toto']['tata']
    assert 1 == deepget(a, 'toto:tata')

    with pytest.raises(KeyError):
        deepget(a, 'toto:titi')


def test_decode():
    from ldap2pg.utils import decode_value

    decoded = {'é': [('é', 0xcafe)], 0xdead: None}
    eacute = 'é'.encode('utf-8')
    encoded = {
        eacute: [(eacute, 0xcafe)],
        0xdead: None,
    }

    assert decoded == decode_value(encoded)


def test_make_map():
    from ldap2pg.utils import make_group_map

    values = dict(v0=0, v1=1)
    groups = dict(g0=['v0'], g1=['v1', 'g0'], g2=['g1'])

    aliases = make_group_map(values, groups)

    wanted = dict(
        v0=['v0'],
        v1=['v1'],
        g0=['v0'],
        g1=['v0', 'v1'],
        g2=['v0', 'v1'],
    )

    assert wanted == aliases
