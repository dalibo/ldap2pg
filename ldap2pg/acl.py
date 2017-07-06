class Acl(object):
    def __init__(self, name, inspect, grant, revoke):
        self.name = name
        self.inspect = inspect
        self.grant = grant
        self.revoke = revoke

    def __str__(self):
        return self.name
