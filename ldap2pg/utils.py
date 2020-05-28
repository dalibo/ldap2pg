from __future__ import unicode_literals

import itertools
import sys
from datetime import timedelta, datetime
from fnmatch import fnmatch
from string import Formatter
import textwrap


PY2 = sys.version_info < (3,)

if PY2:  # pragma: nocover_py3
    string_types = (str, unicode)  # noqa
    unicode = unicode  # noqa
    bytes = str
else:  # pragma: nocover_py2
    string_types = (str,)
    unicode = str
    bytes = bytes  # noqa

try:  # pragma: nocover_py2
    from urllib.parse import urlparse, urlunparse
except ImportError:  # pragma: nocover_py3
    from urlparse import urlparse, urlunparse


__all__ = ['urlparse', 'urlunparse']


class AllDatabases(object):
    # Simple object to represent dbname wildcard.
    def __repr__(self):
        return '__ALL_DATABASES__'


def dedent(s):
    return textwrap.dedent(s).strip()


def lower1(string):
    return string[0].lower() + string[1:]


def match(string, patterns):
    for pattern in patterns:
        if fnmatch(string, pattern):
            return pattern
    return False


class UserError(Exception):
    def __init__(self, message, exit_code=1):
        super(UserError, self).__init__(message)
        self.exit_code = exit_code

    @classmethod
    def wrap(cls, message, exit_code=1):
        message = "\n".join(textwrap.wrap(dedent(message)))
        return cls(message, exit_code)


def deepget(mapping, path):
    """Access deep dict entry."""
    if ':' not in path:
        return mapping[path]
    else:
        key, sub = path.split(':', 1)
        return deepget(mapping[key], sub)


def deepset(mapping, path, value):
    """Define deep entry in dict."""
    if ':' not in path:
        mapping[path] = value
    else:
        key, sub = path.split(':', 1)
        submapping = mapping.setdefault(key, {})
        deepset(submapping, sub, value)


def decode_value(value):
    if isinstance(value, bytes):
        return value.decode('utf-8')
    elif hasattr(value, 'items'):
        return dict([
            (decode_value(k), decode_value(v))
            for k, v in value.items()
        ])
    elif isinstance(value, list):
        return [decode_value(v) for v in value]
    elif isinstance(value, tuple):
        return tuple([decode_value(v) for v in value])
    else:
        return value


def encode_value(value):
    # Encode everyting in value. value can be of any types. Actually, tuple and
    # sets are not preserved.
    if hasattr(value, 'encode'):
        return value.encode('utf-8')
    elif hasattr(value, 'items'):
        return dict(
            (encode_value(k), encode_value(v)) for k, v in value.items())
    elif isinstance(value, list):
        return [encode_value(v) for v in value]
    elif isinstance(value, tuple):
        return tuple([encode_value(v) for v in value])
    else:
        return value


def ensure_unicode(obj):
    if isinstance(obj, unicode):
        return obj
    elif isinstance(obj, bytes):
        return obj.decode('utf-8')
    else:
        try:
            return unicode(obj)
        except UnicodeDecodeError:  # pragma: nocover_py3
            return bytes(obj).decode('utf-8')


def iter_deep_keys(dict_):
    for k, v in dict_.items():
        if hasattr(v, 'items'):
            for kk in iter_deep_keys(v):
                yield '%s:%s' % (k, kk)
        else:
            yield k


def iter_format_fields(strings, split=False):
    formatter = Formatter()
    for string in strings:
        for _, field, _, _ in formatter.parse(string):
            if field is None:
                continue
            field = [
                f for f in field.split('.')
                if '(' not in f and ')' not in f
            ]
            yield field if split else '.'.join(field)


def list_descendant(groups, name):
    # Returns the recursive list of all descendant of name in hierarchy
    # `groups`. `groups` is a flat dict of `groups`
    for child in groups[name]:
        if child in groups:
            for grandchild in list_descendant(groups, child):
                yield grandchild
        else:
            yield child


def make_group_map(values, groups=None):
    # Resolve `groups` including other `groups`, and ungrouped values in a
    # single dict mapping either value name or group name to a list of
    # effective values name.

    groups = groups or {}

    # First, add simple map for value -> value
    aliases = dict((k, [k]) for k in values)
    # Now resolve groups descendant to value list and update map.
    aliases.update(dict(
        (k, sorted(set(list_descendant(groups, k))))
        for k in groups
    ))
    return aliases


def uniq(seq):
    seen = set()
    seen_add = seen.add
    return [x for x in seq if not (x in seen or seen_add(x))]


class FormatList(list):
    @classmethod
    def factory(cls, format_list):
        self = cls()
        for format_ in format_list:
            fields = [
                ('.'.join(f), f[0])
                for f in iter_format_fields([format_], split=True)
            ]
            self.append((format_, fields))
        return self

    def __repr__(self):
        return '[%s]' % (', '.join(self.formats),)

    def expand(self, vars_):
        for format_, fields in self:
            # Expand all combination of values, with only values of this format
            # string.
            vars_subset = dict([
                (v, vars_[v])
                for f, v in fields
            ])
            for items in itertools.product(*vars_subset.values()):
                yield format_.format(**dict(zip(vars_subset.keys(), items)))

    @property
    def formats(self):
        """List plain formats as fed in factory."""
        return [f for f, _ in self]

    @property
    def fields(self):
        """Gather all reference fields in all formats."""
        return [
            field
            for _, fields in self
            for field, _ in fields
        ]

    @property
    def has_static(self):
        return bool([x for x in self if not x[1]])


def collect_fields(*field_lists):
    return set(itertools.chain(*[
        list_.fields for list_ in field_lists
    ]))


def make_format_vars(fields, dn, values):
    # Build variables to inject into format, implementing deep access of
    # compound values like DN and joins. values is a dictionnary with full
    # format token as key and the list of all values available in entries as
    # values. e.g. {'dn.cn': ['toto'], 'dn': ['cn=toto']}.
    vars_ = dict(dn=[dn])
    for field in fields:
        if '.' in field:
            parent, _, child = field.partition('.')
            objects = vars_.setdefault(parent, [
                Settable(_str=dn)
                for v in values[field]
            ])
            for obj, value in zip(objects, values[field]):
                obj.update({child: value})
        else:
            vars_[field] = values[field]
    return vars_


class Settable(object):
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


class Timer(object):
    def __init__(self):
        self.delta = timedelta()

    def __repr__(self):
        return '<%s %s>' % (self.__class__.__name__, self.delta)

    def time_iter(self, iterator):
        while True:
            try:
                with self:
                    item = next(iterator)
                yield item
            except StopIteration:
                break

    def __enter__(self):
        self.start = datetime.utcnow()

    def __exit__(self, *_):
        self.last_delta = datetime.utcnow() - self.start
        self.delta += self.last_delta
        self.start = None
