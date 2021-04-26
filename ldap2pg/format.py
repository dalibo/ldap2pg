import itertools
from copy import deepcopy
from string import Formatter

from .utils import unicode


class FormatSpec(object):
    # A format string to generate e.g. role names.

    def __init__(self, spec):
        self.spec = spec
        self._fields = None

    def __repr__(self):
        return "%s(%r)" % (self.__class__.__name__, self.spec)

    @property
    def fields(self):
        if self._fields is None:
            self._fields = []
            formatter = Formatter()
            for _, field, _, _ in formatter.parse(self.spec):
                if field is None:
                    continue
                path = [
                    f for f in field.split('.')
                    # Exclude method call.
                    if '(' not in f and ')' not in f
                ]
                self._fields.append(
                    FormatField(path[0], '.'.join(path[1:]))
                )
        return self._fields

    @property
    def static(self):
        return 0 == len(self.fields)

    @property
    def attributes_map(self):
        # Aggregate attributes map of all fields.
        map_ = AttributesMap()
        for field in self.fields:
            map_.update(field.attributes_map)
        return map_

    def iter_combinations(self, vars_):
        # Here is the core logic for combination of attributes.
        #
        # vars_ has the same schema as attributes_map. Top level keys define
        # the entry name (either __self__ or a join name). Top level values are
        # list of either string (regular value) or dict (for DN components).
        #
        # First, combine entries, and then, in each entry, combine attributes.
        objcombinations = {}
        map_ = self.attributes_map.intersection(
            getattr(vars_, 'attributes_map', None)
            or AttributesMap.from_dict(vars_)
        )

        for objname, objattrs in map_.items():
            objcombinations[objname] = []
            objattrs = list(set(["dn"]) | objattrs)
            for entry in vars_[objname]:
                subset = dict([
                    (k, v)
                    for k, v in entry.items()
                    if k in objattrs
                ])

                objcombinations[objname].extend([
                    dict(zip(subset.keys(), combination))
                    for combination in
                    itertools.product(*subset.values())
                ])

        for combinations in itertools.product(*objcombinations.values()):
            out = dict()
            for objname, attrs in zip(objcombinations.keys(), combinations):
                attrs = dict([
                    (k, (
                        FormatEntry(_str=v['dn'], **v)
                        if isinstance(v, dict) else
                        v
                    ))
                    for k, v in attrs.items()
                ])
                if '__self__' == objname:
                    out.update(attrs)
                else:
                    out[objname] = FormatEntry(
                        _str=unicode(attrs['dn']), **attrs)
            yield out

    def expand(self, vars_):
        for combination in self.iter_combinations(vars_):
            yield self.spec.format(**combination)


class AttributesMap(dict):
    # Mapping for format variables dict.
    #
    # It's a dictionnary with variable name as key and a set of attributes as
    # value. The variable name are __self__ for accessing LDAPEntry attributes
    # or join attribute name for children entries access.
    #
    # Each FormatSpec is meant to *request* attributes from LDAPEntry or
    # children of it. The fusion of map from all specs of a RoleRule generates
    # the schema of the final big variables dict holding all values for
    # formatting.
    #
    # The schema of two specs on the same entry may have a conflict when using
    # both {member} and {member.cn} in the same role or grant rule. {member} is
    # considered attribute member of __self__ entry while {member.cn} is
    # cn attribute of child member.
    #
    # This will conflict when feeding str.format with member value. Should it
    # be the list of __self__.member or all member objects? The code choose to
    # always use member entry instead of member attribute.
    #
    # When spreading the conflict in different FormatSpec, we can be aware of
    # the conflict by always passing map_ with vars_.

    @classmethod
    def from_dict(cls, dct):
        return cls([
            (k, set(v[0].keys()))
            for k, v in dct.items()
        ])

    @classmethod
    def gather(cls, *maps):
        self = cls()
        for map_ in maps:
            self.update(map_)
        return self

    def __add__(self, other):
        res = self.__class__()
        res.update(self)
        res.update(other)
        return res

    def intersection(self, other):
        i = deepcopy(self)

        for name in list(i.get("__self__", [])):
            if name in other:
                i["__self__"].remove(name)
                i[name] = set(["dn"])

        for name, attrs in list(i.items()):
            if name in other.get("__self__", []):
                i[name] = set(["dn"])
            elif name in other:
                i[name] &= other[name]
            else:
                del i[name]

        return i

    def update(self, other):
        # Merge objects and their attributes
        for objname, attrset in other.items():
            myset = self.setdefault(objname, set())
            myset.update(attrset)

        # Remove joined attribute from self.
        if "__self__" not in self:
            return

        for name in set(self.keys()):
            if name not in self["__self__"]:
                continue

            self["__self__"].remove(name)
            self[name].add("dn")


class FormatField(object):
    # A single {} field from a FormatSpec.

    def __init__(self, var, attribute=None):
        self.var = var
        self.attribute = attribute or None

    def __eq__(self, other):
        return self.as_tuple() == other.as_tuple()

    def __hash__(self):
        return hash(self.as_tuple())

    def __repr__(self):
        return '%s(%r, %r)' % (
            self.__class__.__name__,
            self.var,
            self.attribute,
        )

    def __str__(self):
        return '%s%s' % (
            self.var,
            '.%s' % self.attribute if self.attribute else '',
        )

    def as_tuple(self):
        return self.var, self.attribute

    @property
    def attributes_map(self):
        # Determine to which object the value should be fetched : parent entry
        # (__self__) or a named join entry.
        if self.var == 'dn':
            # {dn.cn} -> __self__."dn.cn"
            object_ = '__self__'
            attribute = str(self)
        elif self.attribute:
            # {member.mail} -> member."mail"
            object_ = self.var
            attribute = self.attribute
        else:
            # {member} -> __self__."member"
            object_ = "__self__"
            attribute = self.var
        return AttributesMap({object_: set([attribute])})


class FormatList(list):
    # A list of format specs

    @classmethod
    def factory(cls, format_list):
        self = cls()
        for format_ in format_list:
            self.append(FormatSpec(format_))
        return self

    def __repr__(self):
        return '[%s]' % (', '.join(self.formats),)

    @property
    def attributes_map(self):
        map_ = AttributesMap()
        for spec in self:
            map_.update(spec.attributes_map)
        return map_

    def expand(self, vars_):
        for spec in self:
            for string in spec.expand(vars_):
                yield string

    @property
    def formats(self):
        """List plain formats as fed in factory."""
        return [spec.spec for spec in self]

    @property
    def fields(self):
        """Gather all reference fields in all formats."""
        return [
            field
            for spec in self
            for field in spec.fields
        ]

    @property
    def has_static(self):
        return bool([x for x in self if x.static])


def collect_fields(*field_lists):
    return set(itertools.chain(*[
        list_.fields for list_ in field_lists
    ]))


class FormatVars(dict):
    # A dictionnary of values from LDAP, grouped for combination, and
    # associated with an Attributes map.
    def __init__(self, map_, *a, **kw):
        self.attributes_map = map_
        super(FormatVars, self).__init__(*a, **kw)


class FormatEntry(object):
    # Object for dot access of attributes in format, like {member.cn}. Allows
    # to render {member} and {member.cn} in the same string.

    def __init__(self, **kw):
        self._str = "**unset**"
        self.update(kw)

    def __repr__(self):
        return '<%s %s>' % (
            self.__class__.__name__,
            ' '.join(['%s=%s' % i for i in self.__dict__.items()])
        )

    def __str__(self):
        return self._str

    def update(self, kw):
        self.__dict__.update(kw)


class FormatValue(object):
    def __init__(self, value):
        self.value = value

    def __str__(self):
        return self.value

    def __repr__(self):
        return 'FormatValue(%r)' % (self.value)

    def __eq__(self, other):
        return self.value == str(other)

    def __getattr__(self, name):
        if name in ['lower()', 'upper()']:
            return getattr(self.value, name[:-2])()
        else:
            raise AttributeError(name)
