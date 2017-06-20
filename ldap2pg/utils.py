import collections


def deepupdate(self, other):
    """Recursive update of two dict."""
    for k, v in other.items():
        if isinstance(v, collections.Mapping):
            r = deepupdate(self.get(k, {}), v)
            self[k] = r
        else:
            self[k] = other[k]
    return self


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
