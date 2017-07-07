from fnmatch import fnmatch


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
