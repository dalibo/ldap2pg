class Query(object):
    def __init__(self, message, rowcount=-1, *args):
        self.message = message
        self.args = args
        self.rowcount = rowcount

    def __str__(self):
        return self.message


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
