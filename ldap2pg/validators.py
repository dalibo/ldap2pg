import logging
from itertools import chain as iterchain

from .format import FormatField
from .ldap import parse_scope
from .ldap import DN_COMPONENTS
from .role import RoleOptions, RoleRule
from .privilege import GrantRule
from .utils import string_types


logger = logging.getLogger(__name__)


default_ldap_query = {
    'base': '',
    'filter': '(objectClass=*)',
    'scope': 'sub',
}

known_ldapsearch_arguments = [
    'allow_missing_attributes',
    'attributes',
    'base',
    'filter',
    'joins',
    'on_unexpected_dn',
    'scope',
]


def ldapquery(value, format_fields=None):
    # ldapquery specs is following.
    #
    # ldapsearch:
    #   base:
    #   filter:
    #   scope:
    #   attributes:
    #   on_unexpected_dn:
    #   allow_missing_attributes:
    #   joins:
    #     member:
    #       base:
    #       filter:
    #       scope:
    #       attributes:
    #
    # Attributes are infered from format_fields. Joins are infered from
    # attributes.

    if not isinstance(value, dict):
        raise ValueError("ldapsearch: is not a dict")

    query = dict(default_ldap_query, **value)
    query['scope'] = parse_scope(query['scope'])
    if 'filter' in query:
        query['filter'] = query['filter'].strip(' \r\n')

    # Accept manual attributes, for legacy. Attribute inference should in all
    # cases. Also, join attributes as predefined this way.
    strlist_alias(query, 'attributes', 'attribute')

    alias(query, 'joins', 'join')
    query.setdefault('joins', {})

    # ATTRIBUTES AND JOINS INFERENCE

    legacy_attrs = query.get('attributes', [])
    format_fields = (
        list(format_fields or []) +
        [FormatField(*a.split('.')) for a in legacy_attrs])

    # Now, loop format fields to list every attributes we need to query from
    # LDAP directory, and for .sub.attr, detect whether we need a subquery (aka
    # join).
    attrs = set()
    for field in format_fields:
        attrs.add(field.var)
        if not field.attribute or field.attribute in DN_COMPONENTS:
            continue
        join = dict(default_ldap_query, **query['joins'].get(field.var, {}))
        join_attrs = join.setdefault('attributes', [])
        join_attrs.append(field.attribute)
        join.setdefault(
            'allow_missing_attributes',
            query.get('allow_missing_attributes', []))
        query['joins'][field.var] = ldapquery(join, [])

    if not attrs:
        fmt = "No attributes are used from LDAP query %(base)s"
        raise ValueError(fmt % value)
    query['attributes'] = list(attrs)

    strlist_alias(query, 'allow_missing_attributes', 'allow_missing_attribute')
    empty_attrs = query.setdefault('allow_missing_attributes', [])
    if 'member' in attrs:
        empty_attrs.append('member')

    # Post process joins.
    for key, join in query['joins'].copy().items():
        if not join.get('attributes'):
            del query['joins'][key]
            continue
        join.pop('base', None)
        join.pop('joins', None)

    # Trim unknown parameters
    for key in list(query.keys()):
        if key not in known_ldapsearch_arguments:
            logger.warning("Ignoring unknown ldapsearch argument '%s'.", key)
            query.pop(key)

    return query


def privilege(raw):
    allowed_keys = set(['grant', 'inspect', 'revoke', 'type'])
    defined_keys = set(raw.keys())
    spurious_keys = defined_keys - allowed_keys

    if spurious_keys:
        msg = "Unknown keys %s" % (', '.join(spurious_keys),)
        raise ValueError(msg)

    raw.setdefault('type', 'nspacl')

    return raw


def privileges(raw):
    if not isinstance(raw, dict):
        raise ValueError('privileges must be a dict')

    value = {}
    for k, v in raw.items():
        if isinstance(v, list):
            value[k] = v
        elif isinstance(v, dict):
            value[k] = privilege(v)
        else:
            msg = "Unknown value %.32s for %s" % (v, k,)
            raise ValueError(msg)
    return value


def rolerule(value):
    rule = value

    if value is None:
        raise ValueError("Empty role rule. Wrong indentation?")

    if isinstance(rule, string_types):
        rule = dict(names=[rule])

    strlist_alias(rule, 'names', 'name')
    compat_ldap_attribute(rule, 'name')
    if 'names' not in rule:
        raise ValueError("Missing role name")

    strlist_alias(rule, 'parents', 'parent')
    compat_ldap_attribute(rule, 'parent')
    rule.setdefault('parents', [])

    strlist_alias(rule, 'members', 'member')
    compat_ldap_attribute(rule, 'member')
    rule.setdefault('members', [])

    options = rule.setdefault('options', {})

    if isinstance(options, string_types):
        options = options.split()

    if isinstance(options, list):
        options = dict(
            (o[2:] if o.startswith('NO') else o, not o.startswith('NO'))
            for o in options
        )

    rule['options'] = RoleOptions(**options)
    return RoleRule(**rule)


def alias(dict_, key, alias):
    if alias in dict_:
        dict_.setdefault(key, dict_.pop(alias))


def strorlist(dict_, key):
    if key in dict_:
        v = dict_[key]
        if isinstance(v, string_types):
            dict_[key] = [v]
        return dict_[key]


def strlist_alias(dict_, key, alias_):
    alias(dict_, key, alias_)
    return strorlist(dict_, key)


def compat_ldap_attribute(rule, name):
    name_attribute = name + '_attribute'
    names = name + 's'
    if name_attribute in rule:
        value = rule.get(names, [])
        rule[names] = value + ['{%s}' % rule.pop(name_attribute)]


def grantrule(value, defaultdb='__all__', defaultschema='__all__'):
    if not isinstance(value, dict):
        raise ValueError('Grant rule must be a dict.')

    alias(value, 'privilege', 'acl')
    compat_ldap_attribute(value, 'role')
    strlist_alias(value, 'roles', 'role')

    value.setdefault('database', defaultdb)
    strlist_alias(value, 'databases', 'database')

    value.setdefault('schema', defaultschema)
    strlist_alias(value, 'schemas', 'schema')

    if 'privilege' not in value:
        raise ValueError('Missing privilege to grant rule.')

    allowed_keys = set([
        'privilege', 'databases', 'schemas',
        'roles', 'role_match',
    ])
    defined_keys = set(value.keys())

    if defined_keys - allowed_keys:
        msg = 'Unknown parameter to grant rules: %s' % (
            ', '.join(defined_keys - allowed_keys)
        )
        raise ValueError(msg)

    if 'roles' not in value:
        raise ValueError('Missing role in grant rule.')

    return GrantRule(**value)


KNOWN_MAPPING_KEYS = set([
    'description',
    'grant',
    'ldap',  # Legacy alias.
    'ldapsearch',
    'role',
    'roles',
])


def ismapping(value):
    # Check whether a YAML value is supposed to be a single mapping.
    if not isinstance(value, dict):
        return False
    return KNOWN_MAPPING_KEYS >= set(value.keys())


def mapping(value, **kw):
    # A single mapping from a query to a set of role rules. This function
    # translate random YAML to cannonical schema.

    if not isinstance(value, dict):
        raise ValueError("Mapping should be a dict.")

    if 'role' in value:
        value['roles'] = value.pop('role')
    if 'roles' not in value:
        value['roles'] = []
    if not isinstance(value['roles'], list):
        value['roles'] = [value['roles']]

    on_unexpected_dn = set([
        r.pop('on_unexpected_dn')
        for r in value['roles']
        if 'on_unexpected_dn' in r
    ])
    if len(on_unexpected_dn) > 1:
        raise ValueError("Mixed on_unexpected_dn not supported.")
    try:
        on_unexpected_dn = next(iter(on_unexpected_dn))
    except StopIteration:
        on_unexpected_dn = 'fail'

    value['roles'] = [rolerule(r) for r in value['roles']]

    if 'grant' in value:
        if isinstance(value['grant'], dict):
            value['grant'] = [value['grant']]
        value['grant'] = [grantrule(g, **kw) for g in value['grant']]

    if not value['roles'] and 'grant' not in value:
        # Don't accept unused LDAP queries.
        raise ValueError("Missing role or grant rule.")

    alias(value, 'ldapsearch', 'ldap')
    if 'ldapsearch' in value:
        roles = value.get('roles', [])
        grants = value.get('grant', [])
        format_fields = set(
            iterchain(*[r.all_fields for r in roles + grants])
        )
        value['ldapsearch'].setdefault('on_unexpected_dn', on_unexpected_dn)
        value['ldapsearch'] = ldapquery(value['ldapsearch'], format_fields)

    return value


def _flatten_legacy_map(value):
    for dbname, dbvalue in value.items():
        if ismapping(dbvalue):
            dbvalue = [dbvalue]

        if isinstance(dbvalue, dict):
            for schema, svalues in dbvalue.items():
                if ismapping(svalues):
                    svalues = [svalues]
                for svalue in svalues:
                    yield mapping(
                        svalue, defaultdb=dbname, defaultschema=schema)
        elif isinstance(dbvalue, list):
            for v in dbvalue:
                yield mapping(v, defaultdb=dbname)


def syncmap(value):
    # Validate and translate raw YAML value to cannonical form used internally.
    #
    # A sync map has the following canonical schema:
    #
    #   - ldapsearch: <ldapquery>
    #     roles:
    #     - <rolerule>
    #     - ...
    #     grant:
    #     - <grantrule>
    #     - ...
    #   ...
    # ...
    #
    # But we accept a wide variety of shorthand schemas:
    #
    # Single mapping:
    #
    # roles: [<rolerule>]
    #
    # List of mapping:
    #
    # - roles: [<rolerule>]
    # - ...
    #
    # dict of dbname->single mapping (legacy)
    #
    # appdb:
    #   roles: <rolerule>
    #
    # dict of dbname->list of mapping (legacy)
    #
    # appdb:
    # - roles: <rolerule>
    #
    # dict of dbname->schema->single mapping (legacy)
    #
    # appdb:
    # - roles: <rolerule>
    # dict of dbname->schema->single mapping (legacy)
    #
    # appdb:
    #   appschema:
    #     roles: <rolerule>

    if not value:
        return []

    if ismapping(value):
        value = [value]

    if isinstance(value, list):
        return [mapping(v) for v in value]
    elif isinstance(value, dict):
        return list(_flatten_legacy_map(value))
    else:
        raise ValueError("Illegal value for sync_map.")


def raw(v):
    return v


def shared_queries(v):
    if not v:
        return {}
    if not hasattr(v, 'items'):
        raise ValueError("shared_queries must be dict.")
    for k, q in v.items():
        if not isinstance(q, string_types):
            raise ValueError("Bad query %s." % k)
    return v


VERBOSITIES = [
    'CRITICAL',
    'ERROR',
    'WARNING',
    'CHANGE',
    'INFO',
    'DEBUG',
]


def verbosity(v):
    if isinstance(v, list):
        v = sum(v)
        v = max(0, v)
        v = min(v, 5)
        v = VERBOSITIES[v]

    if v not in VERBOSITIES:
        raise ValueError("Unknown verbosity '%s'" % (v,))

    return v
