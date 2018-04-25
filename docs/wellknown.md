<!-- GENERATED FROM docs/wellknown.md.j2 -->
<!--*- markdown -*-->

<h1>Well-known Privileges</h1>

`ldap2pg` provides some well-known privileges for recurrent usage. There is **no
warranty** of on these privileges. You have to check privileges configuration on
your databases just like you should do with your own code.

The true added-value of well-known privileges is the `inspect` queries
associated and the boilerplate saved for declaring all `GRANT` queries.


## Using Well-known Privileges

Well-known privilege starts and lasts with `__`. `ldap2pg` [disables
privilege](privileges.md#enabling-privilege) starting with `_`. Thus you have to
include well-known privileges in a group to enable them. If two groups reference
the same privilege, it will be deduplicated, don't worry.

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
  - __all_on_schema__
  - __all_on_tables__

sync_map:
- grant:
    privilege: ddl
    database: mydb
    schema: __all__
    role: admins
```

Well-known privilege name follows the following loose convention:

- `_on_all_tables__` is equivalent to `GRANT ... ON ALL TABLES IN SCHEMA ...`.
- `__default_...` is equivalent to `ALTER DEFAULT PRIVILEGES ... IN SCHEMA ...`.
- `__..._on_tables__` groups `__..._on_all_tables__` and
  `__default_..._on_tables__`.
- Group starting with `__all_on_` is *equivalent* to `ALL PRIVILEGES` in SQL.
- A privilege specific to one type does not have `_on_type__` e.g.
  `__delete_on_tables__` is shorten to `__delete__` .


## Privilege Groups

Next is an extensive, boring, list of all well known privilege groups in
`master`. Each group is documented by its name and the list of included
privilege. Each privilege name point the the detail of privilege definition.

Actually, a group like `__all_on_tables__` is implemented as group of groups.
But for the sake of simplicity, the documentation lists the resolved list of
concrete privileges finally included.

Here we go.


<a name="all-on-schemas"></a>
### Group `__all_on_schemas__`

- [`__create_on_schemas__`](#create-on-schemas)
- [`__usage_on_schemas__`](#usage-on-schemas)


<a name="all-on-sequences"></a>
### Group `__all_on_sequences__`

- [`__default_select_on_sequences__`](#default-select-on-sequences)
- [`__default_update_on_sequences__`](#default-update-on-sequences)
- [`__default_usage_on_sequences__`](#default-usage-on-sequences)
- [`__select_on_all_sequences__`](#select-on-all-sequences)
- [`__update_on_all_sequences__`](#update-on-all-sequences)
- [`__usage_on_all_sequences__`](#usage-on-all-sequences)


<a name="all-on-tables"></a>
### Group `__all_on_tables__`

- [`__default_delete_on_tables__`](#default-delete-on-tables)
- [`__default_insert_on_tables__`](#default-insert-on-tables)
- [`__default_references_on_tables__`](#default-references-on-tables)
- [`__default_select_on_tables__`](#default-select-on-tables)
- [`__default_trigger_on_tables__`](#default-trigger-on-tables)
- [`__default_truncate_on_tables__`](#default-truncate-on-tables)
- [`__default_update_on_tables__`](#default-update-on-tables)
- [`__delete_on_all_tables__`](#delete-on-all-tables)
- [`__insert_on_all_tables__`](#insert-on-all-tables)
- [`__references_on_all_tables__`](#references-on-all-tables)
- [`__select_on_all_tables__`](#select-on-all-tables)
- [`__trigger_on_all_tables__`](#trigger-on-all-tables)
- [`__truncate_on_all_tables__`](#truncate-on-all-tables)
- [`__update_on_all_tables__`](#update-on-all-tables)


<a name="delete"></a>
### Group `__delete__`

- [`__default_delete_on_tables__`](#default-delete-on-tables)
- [`__delete_on_all_tables__`](#delete-on-all-tables)


<a name="delete-on-tables"></a>
### Group `__delete_on_tables__`

- [`__default_delete_on_tables__`](#default-delete-on-tables)
- [`__delete_on_all_tables__`](#delete-on-all-tables)


<a name="execute"></a>
### Group `__execute__`

- [`__default_execute_on_functions__`](#default-execute-on-functions)
- [`__execute_on_all_functions__`](#execute-on-all-functions)
- [`__global_default_execute_on_functions__`](#global-default-execute-on-functions)


<a name="execute-on-functions"></a>
### Group `__execute_on_functions__`

- [`__default_execute_on_functions__`](#default-execute-on-functions)
- [`__execute_on_all_functions__`](#execute-on-all-functions)
- [`__global_default_execute_on_functions__`](#global-default-execute-on-functions)


<a name="insert"></a>
### Group `__insert__`

- [`__default_insert_on_tables__`](#default-insert-on-tables)
- [`__insert_on_all_tables__`](#insert-on-all-tables)


<a name="insert-on-tables"></a>
### Group `__insert_on_tables__`

- [`__default_insert_on_tables__`](#default-insert-on-tables)
- [`__insert_on_all_tables__`](#insert-on-all-tables)


<a name="references"></a>
### Group `__references__`

- [`__default_references_on_tables__`](#default-references-on-tables)
- [`__references_on_all_tables__`](#references-on-all-tables)


<a name="references-on-tables"></a>
### Group `__references_on_tables__`

- [`__default_references_on_tables__`](#default-references-on-tables)
- [`__references_on_all_tables__`](#references-on-all-tables)


<a name="select-on-sequences"></a>
### Group `__select_on_sequences__`

- [`__default_select_on_sequences__`](#default-select-on-sequences)
- [`__select_on_all_sequences__`](#select-on-all-sequences)


<a name="select-on-tables"></a>
### Group `__select_on_tables__`

- [`__default_select_on_tables__`](#default-select-on-tables)
- [`__select_on_all_tables__`](#select-on-all-tables)


<a name="trigger"></a>
### Group `__trigger__`

- [`__default_trigger_on_tables__`](#default-trigger-on-tables)
- [`__trigger_on_all_tables__`](#trigger-on-all-tables)


<a name="trigger-on-tables"></a>
### Group `__trigger_on_tables__`

- [`__default_trigger_on_tables__`](#default-trigger-on-tables)
- [`__trigger_on_all_tables__`](#trigger-on-all-tables)


<a name="truncate"></a>
### Group `__truncate__`

- [`__default_truncate_on_tables__`](#default-truncate-on-tables)
- [`__truncate_on_all_tables__`](#truncate-on-all-tables)


<a name="truncate-on-tables"></a>
### Group `__truncate_on_tables__`

- [`__default_truncate_on_tables__`](#default-truncate-on-tables)
- [`__truncate_on_all_tables__`](#truncate-on-all-tables)


<a name="update-on-sequences"></a>
### Group `__update_on_sequences__`

- [`__default_update_on_sequences__`](#default-update-on-sequences)
- [`__update_on_all_sequences__`](#update-on-all-sequences)


<a name="update-on-tables"></a>
### Group `__update_on_tables__`

- [`__default_update_on_tables__`](#default-update-on-tables)
- [`__update_on_all_tables__`](#update-on-all-tables)


<a name="usage-on-sequences"></a>
### Group `__usage_on_sequences__`

- [`__default_usage_on_sequences__`](#default-usage-on-sequences)
- [`__usage_on_all_sequences__`](#usage-on-all-sequences)


## Single Privileges

Next is the list of well-known privileg. Each is associated with a `REVOKE`
query and an `inspect` query implementing full inspection of grantees, including
built-in grants to PUBLIC.

For the actual meaning of each SQL privileges, refer to official [Postgres
documentation of
`GRANT`](https://www.postgresql.org/docs/current/static/sql-grant.html)
statement.



<a name="connect"></a>
### Privilege `__connect__`

``` SQL
GRANT CONNECT ON DATABASE {database} TO {role};
```


<a name="create-on-schemas"></a>
### Privilege `__create_on_schemas__`

``` SQL
GRANT CREATE ON SCHEMA {schema} TO {role};
```


<a name="default-delete-on-tables"></a>
### Privilege `__default_delete_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT DELETE ON TABLES TO {role};
```


<a name="default-execute-on-functions"></a>
### Privilege `__default_execute_on_functions__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT EXECUTE ON FUNCTIONS TO {role};
```


<a name="default-insert-on-tables"></a>
### Privilege `__default_insert_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT INSERT ON TABLES TO {role};
```


<a name="default-references-on-tables"></a>
### Privilege `__default_references_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT REFERENCES ON TABLES TO {role};
```


<a name="default-select-on-sequences"></a>
### Privilege `__default_select_on_sequences__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT SELECT ON SEQUENCES TO {role};
```


<a name="default-select-on-tables"></a>
### Privilege `__default_select_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT SELECT ON TABLES TO {role};
```


<a name="default-trigger-on-tables"></a>
### Privilege `__default_trigger_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT TRIGGER ON TABLES TO {role};
```


<a name="default-truncate-on-tables"></a>
### Privilege `__default_truncate_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT TRUNCATE ON TABLES TO {role};
```


<a name="default-update-on-sequences"></a>
### Privilege `__default_update_on_sequences__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT UPDATE ON SEQUENCES TO {role};
```


<a name="default-update-on-tables"></a>
### Privilege `__default_update_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT UPDATE ON TABLES TO {role};
```


<a name="default-usage-on-sequences"></a>
### Privilege `__default_usage_on_sequences__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT USAGE ON SEQUENCES TO {role};
```


<a name="delete-on-all-tables"></a>
### Privilege `__delete_on_all_tables__`

``` SQL
GRANT DELETE ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="execute-on-all-functions"></a>
### Privilege `__execute_on_all_functions__`

``` SQL
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA {schema} TO {role}
```


<a name="global-default-execute-on-functions"></a>
### Privilege `__global_default_execute_on_functions__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} GRANT EXECUTE ON FUNCTIONS TO {role};
```


<a name="insert-on-all-tables"></a>
### Privilege `__insert_on_all_tables__`

``` SQL
GRANT INSERT ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="references-on-all-tables"></a>
### Privilege `__references_on_all_tables__`

``` SQL
GRANT REFERENCES ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="select-on-all-sequences"></a>
### Privilege `__select_on_all_sequences__`

``` SQL
GRANT SELECT ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```


<a name="select-on-all-tables"></a>
### Privilege `__select_on_all_tables__`

``` SQL
GRANT SELECT ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="temporary"></a>
### Privilege `__temporary__`

``` SQL
GRANT TEMPORARY ON DATABASE {database} TO {role};
```


<a name="trigger-on-all-tables"></a>
### Privilege `__trigger_on_all_tables__`

``` SQL
GRANT TRIGGER ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="truncate-on-all-tables"></a>
### Privilege `__truncate_on_all_tables__`

``` SQL
GRANT TRUNCATE ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="update-on-all-sequences"></a>
### Privilege `__update_on_all_sequences__`

``` SQL
GRANT UPDATE ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```


<a name="update-on-all-tables"></a>
### Privilege `__update_on_all_tables__`

``` SQL
GRANT UPDATE ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="usage-on-all-sequences"></a>
### Privilege `__usage_on_all_sequences__`

``` SQL
GRANT USAGE ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```


<a name="usage-on-schemas"></a>
### Privilege `__usage_on_schemas__`

``` SQL
GRANT USAGE ON SCHEMA {schema} TO {role};
```


<a name="usage-on-types"></a>
### Privilege `__usage_on_types__`

``` SQL
GRANT USAGE ON SCHEMA {schema} TO {role};
```

