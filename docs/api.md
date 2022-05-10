---
hide:
  - navigation
---

<h1>Using ldap2pg as a Python API</h1>

`ldap2pg` Python package exposes a simple API to execute ldap2pg business code
from you own Python script.

``` python
from textwrap import dedent
from ldap2pg import synchronize, UserError

try:
    synchronize(dedent("""\
    sync_map:
    - role:
        name: myrole
    """))
except UserError as e:
    logger.error("%s", e)
```


## ldap2pg.synchronize

``` python
def synchronize(config, environ=None, argv=None):
```

Synchronizes a Postgres cluster from an LDAP directory according to the
configuration described in `config`.

`config` is either a raw YAML document or a Python dict following ldap2pg YAML
format.

`environ` is a dict allowing to override `os.environ`. Likewise, `argv` is a
list of strings overriding `sys.argv`. `argv` is passed as-is to
`argparse.ArgumentParser.parse_args()`.

`synchronize()` returns 0 on success or raises `ldap2pg.UserError` on failure.
If `config['check']` is `True`, `synchronize()` returns the number of queries
generated to synchronize the Postgres cluster.

Any exception other than `ldap2pg.UserError` is an unhandled error and should
be reported as a bug upstream.


## ldap2pg.UserError

``` python
class UserError:
```

Represents an error in environment, configuration or runtime. Attribute
`exit_code` suggests a UNIX process exit code.


## Logging

`synchronize()` does not modify logging configuration. All configurations
options relative to logging are useless when using API. However, ldap2pg makes
heavy usage of logging.

ldap2pg adds a custom logging level named `CHANGE` which is just above `INFO`
level. At import time, ldap2pg register this logging level. `logging` default
logger class is respected and preserved.
