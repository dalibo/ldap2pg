def test_attribute_map():
    from ldap2pg.format import AttributesMap

    map0 = AttributesMap({
        "__self__": set(["cn", "member"]),
    })
    map1 = AttributesMap({
        "member": set(["cn", "mail"]),
    })

    global_map = map0 + map1 + AttributesMap({'extra': set(['cn'])})

    assert "__self__" in global_map
    assert "member" not in global_map["__self__"]
    assert "cn" in global_map["__self__"]
    assert "member" in global_map
    assert "dn" in global_map["member"]
    assert "mail" in global_map["member"]

    reduced = global_map.intersection(map0)

    assert "member" in reduced
    assert "member" not in reduced["__self__"]
    assert "mail" not in reduced["member"]
    assert 'extra' not in reduced

    reversed_ = map0.intersection(global_map)
    assert reduced == reversed_


def test_entry():
    from ldap2pg.format import FormatEntry

    entry = FormatEntry(_str='**str**', key='value')
    assert 'key' in repr(entry)
    assert '**str**' == str(entry)

    entry.update(dict(other='value'))
    assert 'other' in repr(entry)
    assert 'value' == entry.other


def test_field():
    from ldap2pg.format import FormatField

    a = FormatField('member', 'mail')

    assert """FormatField('member', 'mail')""" == repr(a)
    assert 'member.mail' == str(a)


def test_format_list():
    from ldap2pg.format import FormatList

    lst = FormatList.factory([
        'static',
        '{cn}',
        '{dn.cn}',
        '{member}',
        '{member.cn}',
        '{member.mail}',
        '{member.dn.cn}',
    ])

    assert lst.has_static is True
    assert {
        '__self__': set([
            'cn',
            'dn.cn',
        ]),
        'member': set([
            'cn',
            'dn',
            'dn.cn',
            'mail',
        ]),
    } == lst.attributes_map


def test_combinations_self():
    from ldap2pg.format import FormatSpec

    spec = FormatSpec("{cn} {member}")
    assert "{cn} {member}" in repr(spec)

    vars_ = {
        "__self__": [{
            "dn": ["cn=cn0,ou=group"],
            "cn": ["cn0"],
            "member": ["m0", "m1"]
        }],
    }

    combs = list(spec.iter_combinations(vars_))
    assert "cn0" == combs[0]['cn']
    assert "m0" == combs[0]['member']
    assert "cn0" == combs[1]['cn']
    assert "m1" == combs[1]['member']

    assert 2 == len(combs)


def test_combinations_join():
    from ldap2pg.format import FormatSpec

    spec = FormatSpec("{cn} {member} {member.mail}")
    vars_ = {
        "__self__": [{
            "dn": ["cn=cn0,ou=group"],
            "cn": ["cn0"],
        }],
        "member": [
            {
                "dn": ["cn=m0,ou=member"],
                "mail": ["m0@toto", "m00@toto"],
            },
            {
                "dn": ["cn=m1,ou=member"],
                "mail": ["m1@toto"],
            },
        ],
    }

    combs = list(spec.iter_combinations(vars_))
    assert 3 == len(combs)
    assert "cn0" == combs[0]['cn']
    assert "m0@toto" == combs[0]['member'].mail
    assert "cn0" == combs[1]['cn']
    assert "m00@toto" == combs[1]['member'].mail
    assert "cn0" == combs[2]['cn']
    assert "m1@toto" == combs[2]['member'].mail

    spec = FormatSpec("{cn} {member}")
    combs = list(spec.iter_combinations(vars_))
    assert 2 == len(combs)
    assert "cn=m0,ou=member" == combs[0]["member"]._str
    assert "cn=m1,ou=member" == combs[1]["member"]._str


def test_format():
    from ldap2pg.format import FormatList

    vars_ = {
        "__self__": [{
            "dn": [dict(
                dn="cn=cn0,ou=group",
                cn="cn0",
            )],
            "cn": ["cn0"],
        }],
        "member": [
            {
                "dn": [dict(dn="cn=m0,ou=member", cn="m0")],
                "cn": ["m0"],
                "mail": ["m0@toto", "m00@toto"],
            },
            {
                "dn": [dict(dn="cn=m1,ou=member", cn="m1")],
                "cn": ["m1"],
                "mail": ["m1@toto"],
            },
        ],
    }

    flist = FormatList.factory([
        '{cn}', '{dn.cn}',
        '{member}', '{member.cn}: {member.mail}', '{member.dn.cn}',
    ])

    values = flist.expand(vars_)

    wanted = [
        "cn0",
        "cn0",
        "cn=m0,ou=member",
        "cn=m1,ou=member",
        "m0: m0@toto",
        "m0: m00@toto",
        "m1: m1@toto",
        "m0",
        "m1",
    ]

    assert wanted == list(values)
