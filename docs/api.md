---
hide:
  - navigation
---

<h1>Use ldap2pg as a Python API</h1>

ldap2pg Python package exposes a simple API to execute ldap2pg business code
from you own Python script.

```
synchronize(config, environ=None, argv=None)
```

Synchronize a Postgres cluster from an LDAP directory according to the
configuration described in `config`.

`config` is either a raw YAML document or a Python dict following ldap2pg YAML
format.

`environ` is a dict allowing to override `os.environ`. Likewise, `argv` is a
list of string overriding `sys.argv`. `argv` is passed as-is to
`argparse.ArgumentParser.parse_args()`.

`synchronize()` returns 0 on success or raises `ldap2pg.UserError` on failure.
If `config['check']` is True, `synchronize()` returns the number of queries
generated to synchronize the Postgres cluster.

Any exception other than `ldap2pg.UserError` is an unhandled error and should
be reported as a bug upstream.

```
exception UserError
```

Represent an error in environment or configuration. Attribute `exit_code`
suggests a UNIX process exit code.


## Logging

`synchronize()` does not modify logging configuration. All configurations
options relative to logging are useless when using API. However, ldap2pg makes
heavy usage of logging.

ldap2pg adds a custom logging level named `CHANGE` which is just above `INFO`
level. At import time, ldap2pg register this logging level. `logging` default
logger class is respected and preserved.


## Example

``` python
from textwrap import dedent
from ldap2pg import synchronize, UserError

try:
    synchronize(dedent("""\
    dry: false

    sync_map:
    - role: myrole
    """))
except UserError as e:
    logger.exception("ldap2pg failed:")
    exit(e.exit_code)
```
