<!--*- markdown -*-->

<h1>Builtins Privileges</h1>

ldap2pg provides some builtin ACL and predefined privilege profiles for recurrent usage.
There is **no warranty** on these privileges.
You have to check privileges configuration on your databases just like you should do with your own code.


## Using Predefined Privilege Profiles

A privilege profile is a list of reference to a privilege type in an ACL.
In ldap2pg, an ACL is a set of query to inspect, grant and revoke privilege on a class of objects.
The inspect query expands `aclitem` PostgreSQL type to list all grants from system catalog.
Privilege profile can include another profile.

Builtin privilege profile starts and ends with `__`.
ldap2pg [disables privilege profile](config.md#privileges) starting with `_`.
Thus you have to include builtin privileges profile in another profile to enable them.
If two profiles reference the same privilege, ldap2pg will inspect it once.

``` yaml
privileges:
  ro:
  - __connect__
  - __usage_on_schemas__
  - __select_on_tables__

  rw:
  - ro
  - __insert__
  - __update_on_tables__

  ddl:
  - rw
  - __all_on_schemas__
  - __all_on_tables__

rules:
- grant:
    privilege: ddl
    database: mydb
    role: admins
```

Builtin profile's name follows the following loose convention:

- `..._on_all_tables__` references `ALL TABLES IN SCHEMA` ACL.
   Likewise for sequences and functions.
- `__default_...__` references both global and schema-wide default privileges.
- `__..._on_tables__` groups `__..._on_all_tables__` and `__default_..._on_tables__`.
- Group starting with `__all_on_...__` is *equivalent* to `ALL PRIVILEGES` in SQL.
  However, each privilege will be granted individually.
- A privilege specific to one object type does not have `_on_<type>` suffix.
  E.g. `__delete_on_tables__` is aliased to `__delete__`.

This page does not document the SQL standard and the meaning of each SQL privileges.
You will find the documentation of SQL privileges in [Postgresql GRANT documentation] and [ALTER DEFAULT PRIVILEGES documentation].

[Postgresql GRANT documentation]: https://www.postgresql.org/docs/current/sql-grant.html
[ALTER DEFAULT PRIVILEGES documentation]: https://www.postgresql.org/docs/current/sql-alterdefaultprivileges.html


## ACL Reference

Here is the list of builtin ACL.

For effective privileges:

- `DATABASE`: privilege on database like `CONNECT`, `CREATE`, etc.
- `LANGUAGE`: manage `USAGE` on procedural languages.
- `ALL FUNCTIONS IN SCHEMA`: manage `EXECUTE` on all functions per schema.
- `ALL SEQUENCES IN SCHEMA`: like above but for sequences.
- `ALL TABLES IN SCHEMA`: like above but for tables and views.

`ALL ... IN SCHEMA` ACL inspects whether a privilege is granted to only a subset of objects.
This is a *partial* grant.
A partial grant is either revoked if unwanted or regranted if expected.

ACL for default privileges:

- `SEQUENCES`
- `FUNCTIONS`
- `TABLES`

Theses ACL must be referenced with `global` set to either `schema` or `global`.

You can reference these ACL using [privileges:on] parameter in YAML. Like this:

``` yaml
privileges:
  myprofile:
  - type: SELECT
    on: ALL TABLES IN SCHEMA
```

[privileges:on]: config.md#privileges-on

You cannot (yet) configure custom ACL.


## Profiles Reference

### Profile `__all_on_functions__` { #all-on-functions  data-toc-label="&#95;&#95;all&#95;on&#95;functions&#95;&#95;" }

- [`__execute_on_functions__`](#execute-on-functions)


### Profile `__all_on_schemas__` { #all-on-schemas  data-toc-label="&#95;&#95;all&#95;on&#95;schemas&#95;&#95;" }

- [`__create_on_schemas__`](#create-on-schemas)
- [`__usage_on_schema__`](#usage-on-schema)


### Profile `__all_on_sequences__` { #all-on-sequences  data-toc-label="&#95;&#95;all&#95;on&#95;sequences&#95;&#95;" }

- [`__select_on_sequences__`](#select-on-sequences)
- [`__update_on_sequences__`](#update-on-sequences)
- [`__usage_on_sequences__`](#usage-on-sequences)


### Profile `__all_on_tables__` { #all-on-tables  data-toc-label="&#95;&#95;all&#95;on&#95;tables&#95;&#95;" }

- [`__delete_on_tables__`](#delete-on-tables)
- [`__insert_on_tables__`](#insert-on-tables)
- [`__select_on_tables__`](#select-on-tables)
- [`__truncate_on_tables__`](#truncate-on-tables)
- [`__update_on_tables__`](#update-on-tables)
- [`__references_on_tables__`](#references-on-tables)
- [`__trigger_on_tables__`](#trigger-on-tables)


### Profile `__delete_on_tables__` { #delete-on-tables  data-toc-label="&#95;&#95;delete&#95;on&#95;tables&#95;&#95;" }

- [`__default_delete_on_tables__`](#default-delete-on-tables)
- [`__delete_on_all_tables__`](#delete-on-all-tables)


### Profile `__execute_on_functions__` { #execute-on-functions  data-toc-label="&#95;&#95;execute&#95;on&#95;functions&#95;&#95;" }

- [`__default_execute_on_functions__`](#default-execute-on-functions)
- [`__execute_on_all_functions__`](#execute-on-all-functions)


### Profile `__insert_on_tables__` { #insert-on-tables  data-toc-label="&#95;&#95;insert&#95;on&#95;tables&#95;&#95;" }

- [`__default_insert_on_tables__`](#default-insert-on-tables)
- [`__insert_on_all_tables__`](#insert-on-all-tables)


### Profile `__references_on_tables__` { #references-on-tables  data-toc-label="&#95;&#95;references&#95;on&#95;tables&#95;&#95;" }

- [`__default_references_on_tables__`](#default-references-on-tables)
- [`__references_on_all_tables__`](#references-on-all-tables)


### Profile `__select_on_sequences__` { #select-on-sequences  data-toc-label="&#95;&#95;select&#95;on&#95;sequences&#95;&#95;" }

- [`__default_select_on_sequences__`](#default-select-on-sequences)
- [`__select_on_all_sequences__`](#select-on-all-sequences)


### Profile `__select_on_tables__` { #select-on-tables  data-toc-label="&#95;&#95;select&#95;on&#95;tables&#95;&#95;" }

- [`__default_select_on_tables__`](#default-select-on-tables)
- [`__select_on_all_tables__`](#select-on-all-tables)


### Profile `__trigger_on_tables__` { #trigger-on-tables  data-toc-label="&#95;&#95;trigger&#95;on&#95;tables&#95;&#95;" }

- [`__default_trigger_on_tables__`](#default-trigger-on-tables)
- [`__trigger_on_all_tables__`](#trigger-on-all-tables)


### Profile `__truncate_on_tables__` { #truncate-on-tables  data-toc-label="&#95;&#95;truncate&#95;on&#95;tables&#95;&#95;" }

- [`__default_truncate_on_tables__`](#default-truncate-on-tables)
- [`__truncate_on_all_tables__`](#truncate-on-all-tables)


### Profile `__update_on_sequences__` { #update-on-sequences  data-toc-label="&#95;&#95;update&#95;on&#95;sequences&#95;&#95;" }

- [`__default_update_on_sequences__`](#default-update-on-sequences)
- [`__update_on_all_sequences__`](#update-on-all-sequences)


### Profile `__update_on_tables__` { #update-on-tables  data-toc-label="&#95;&#95;update&#95;on&#95;tables&#95;&#95;" }

- [`__default_update_on_tables__`](#default-update-on-tables)
- [`__update_on_all_tables__`](#update-on-all-tables)


### Profile `__usage_on_sequences__` { #usage-on-sequences  data-toc-label="&#95;&#95;usage&#95;on&#95;sequences&#95;&#95;" }

- [`__default_usage_on_sequences__`](#default-usage-on-sequences)
- [`__usage_on_all_sequences__`](#usage-on-all-sequences)


## Privileges Reference

Here is the list of predefined privileges:

| Name | Manages |
|------|---------|
| `__connect__`                            | `CONNECT ON DATABASE` |
| `__create_on_schemas__`                  | `CREATE ON SCHEMA` |
| `__delete_on_all_tables__`               | `DELETE ON ALL TABLES IN SCHEMA` |
| `__execute_on_all_functions__`           | `EXECUTE ON ALL FUNCTIONS IN SCHEMA` |
| `__insert_on_all_tables__`               | `INSERT ON ALL TABLES IN SCHEMA` |
| `__references_on_all_tables__`           | `REFERENCES ON ALL TABLES IN SCHEMA` |
| `__select_on_all_sequences__`            | `SELECT ON ALL SEQUENCES IN SCHEMA` |
| `__select_on_all_tables__`               | `SELECT ON ALL TABLES IN SCHEMA` |
| `__temporary__`                          | `TEMPORARY ON DATABASE` |
| `__trigger_on_all_tables__`              | `TRIGGER ON ALL TABLES IN SCHEMA` |
| `__truncate_on_all_tables__`             | `TRUNCATE ON ALL TABLES IN SCHEMA` |
| `__update_on_all_sequences__`            | `UPDATE ON ALL SEQUENCES IN SCHEMA` |
| `__update_on_all_tables__`               | `UPDATE ON ALL TABLES IN SCHEMA` |
| `__usage_on_all_sequences__`             | `USAGE ON ALL SEQUENCES IN SCHEMA` |
| `__usage_on_schemas__`                   | `USAGE ON SCHEMA` |



## Default Privileges Reference

Here is the list of predefined default privileges.
Default privilege profile references both global and schema defaults.

| Name | Manages |
|------|---------|
| <a name="default-delete-on-tables"></a> `__default_delete_on_tables__`           | `DELETE ON TABLES` |
| <a name="default-execute-on-functions"></a> `__default_execute_on_functions__`       | `EXECUTE ON FUNCTIONS` |
| <a name="default-insert-on-tables"></a> `__default_insert_on_tables__`           | `INSERT ON TABLES` |
| <a name="default-references-on-tables"></a> `__default_references_on_tables__`       | `REFERENCES ON TABLES` |
| <a name="default-select-on-sequences"></a> `__default_select_on_sequences__`        | `SELECT ON SEQUENCES` |
| <a name="default-select-on-tables"></a> `__default_select_on_tables__`           | `SELECT ON TABLES` |
| <a name="default-trigger-on-tables"></a> `__default_trigger_on_tables__`          | `TRIGGER ON TABLES` |
| <a name="default-truncate-on-tables"></a> `__default_truncate_on_tables__`         | `TRUNCATE ON TABLES` |
| <a name="default-update-on-sequences"></a> `__default_update_on_sequences__`        | `UPDATE ON SEQUENCES` |
| <a name="default-update-on-tables"></a> `__default_update_on_tables__`           | `UPDATE ON TABLES` |
| <a name="default-usage-on-sequences"></a> `__default_usage_on_sequences__`         | `USAGE ON SEQUENCES` |
