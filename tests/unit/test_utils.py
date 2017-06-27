from __future__ import unicode_literals

import pytest


def test_query():
    from ldap2pg.utils import Query

    qry = Query('message', 1, 'SELECT %s;', ('args',))

    assert 1 == qry.rowcount
    assert 2 == len(qry.args)
    assert 'message' == str(qry)


def test_deep_getset():
    from ldap2pg.utils import deepget, deepset

    a = dict()

    deepset(a, 'toto:tata', 1)

    assert 1 == a['toto']['tata']
    assert 1 == deepget(a, 'toto:tata')

    with pytest.raises(KeyError):
        deepget(a, 'toto:titi')
