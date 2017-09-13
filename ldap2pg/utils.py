from __future__ import unicode_literals

import sys
from fnmatch import fnmatch


PY2 = sys.version_info < (3,)

if PY2:
    string_types = (str, unicode)  # noqa
else:
    string_types = (str,)


class AllDatabases(object):
    # Simple object to represent dbname wildcard.
    def __repr__(self):
        return '__ALL_DATABASES__'


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
    aliases = {k: [k] for k in values}
    # Now resolve groups descendant to value list and update map.
    aliases.update({
        k: sorted(list_descendant(groups, k))
        for k in groups
    })
    return aliases
