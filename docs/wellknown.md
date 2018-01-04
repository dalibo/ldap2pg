<!-- GENERATED FROM docs/wellknown.md.j2 -->
<!--*- markdown -*-->

<h1>Well-known ACL</h1>

`ldap2pg` provides some well-known ACLs for recurrent usage. There is **no
warranty** of on these ACLs. You have to check privileges configuration on your
databases just like you should do with your own code.

The true added-value of well-known ACLs is the `inspect` queries associated and
the boilerplate saved for declaring all `GRANT` queries.


## Using Well-known ACLs

Well-known ACL starts and lasts with `__`. `ldap2pg` [disables
ACL](acl.md#enabling-acl) starting with `_`. Thus you have to include well-known
ACLs in a group to enable them. If two groups reference the same ACL, it will be
deduplicated, don't worry.

``` yaml
acls:
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
    acl: ddl
    database: mydb
    schema: __all__
    role: admins
```

Well-known ACL name follows the following loose convention:

- `_on_all_tables__` is equivalent to `GRANT ... ON ALL TABLES IN SCHEMA ...`.
- `__default_...` is equivalent to `ALTER DEFAULT PRIVILEGES ... IN SCHEMA ...`.
- `__..._on_tables__` groups `__..._on_all_tables__` and
  `__default_..._on_tables__`.
- Group starting with `__all_on_` is *equivalent* to `ALL PRIVILEGES` in SQL.
- A privilege specific to one type does not have `_on_type__` e.g.
  `__delete_on_tables__` is shorten to `__delete__` .


## ACL groups

Next is an extensive, boring, list of all well known ACL groups in `master`.
Each group is documented by its name and the list of included ACL. Each ACL name
point the the detail of ACL definition.

Actually, a group like `__all_on_tables__` is implemented as group of groups.
But for the sake of simplicity, the documentation lists the resolved list of
concrete ACLs finally included.

Here we go.


<a name="all-on-schemas"></a>
### Group `__all_on_schemas__`

- [`__create_on_schemas__`](#create-on-schemas)
- [`__default_usage_on_sequences__`](#default-usage-on-sequences)
- [`__usage_on_all_sequences__`](#usage-on-all-sequences)


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


<a name="execute"></a>
### Group `__execute__`

- [`__default_execute_on_functions__`](#default-execute-on-functions)
- [`__execute_on_all_functions__`](#execute-on-all-functions)


<a name="insert"></a>
### Group `__insert__`

- [`__default_insert_on_tables__`](#default-insert-on-tables)
- [`__insert_on_all_tables__`](#insert-on-all-tables)


<a name="references"></a>
### Group `__references__`

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


<a name="truncate"></a>
### Group `__truncate__`

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


## ACLs

Next is the list of well-known ACL. Each is associated with a `REVOKE` query and
an `inspect` query implementing full inspection of grantees, including built-in
grants to PUBLIC.

For the actual meaning of each SQL privileges, refer to official [Postgres
documentation of
`GRANT`](https://www.postgresql.org/docs/current/static/sql-grant.html)
statement.



<a name="connect"></a>
### ACL `__connect__`

``` SQL
GRANT CONNECT ON DATABASE {database} TO {role};
```


<a name="create-on-schemas"></a>
### ACL `__create_on_schemas__`

``` SQL
GRANT CREATE ON SCHEMA {schema} TO {role};
```


<a name="default-delete-on-tables"></a>
### ACL `__default_delete_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT DELETE ON TABLES TO {role};
```


<a name="default-execute-on-functions"></a>
### ACL `__default_execute_on_functions__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT EXECUTE ON FUNCTIONS TO {role};
```


<a name="default-insert-on-tables"></a>
### ACL `__default_insert_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT INSERT ON TABLES TO {role};
```


<a name="default-references-on-tables"></a>
### ACL `__default_references_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT REFERENCES ON TABLES TO {role};
```


<a name="default-select-on-sequences"></a>
### ACL `__default_select_on_sequences__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT SELECT ON SEQUENCES TO {role};
```


<a name="default-select-on-tables"></a>
### ACL `__default_select_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT SELECT ON TABLES TO {role};
```


<a name="default-trigger-on-tables"></a>
### ACL `__default_trigger_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT TRIGGER ON TABLES TO {role};
```


<a name="default-truncate-on-tables"></a>
### ACL `__default_truncate_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT TRUNCATE ON TABLES TO {role};
```


<a name="default-update-on-sequences"></a>
### ACL `__default_update_on_sequences__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT UPDATE ON SEQUENCES TO {role};
```


<a name="default-update-on-tables"></a>
### ACL `__default_update_on_tables__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT UPDATE ON TABLES TO {role};
```


<a name="default-usage-on-sequences"></a>
### ACL `__default_usage_on_sequences__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT USAGE ON SEQUENCES TO {role};
```


<a name="delete-on-all-tables"></a>
### ACL `__delete_on_all_tables__`

``` SQL
GRANT DELETE ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="execute-on-all-functions"></a>
### ACL `__execute_on_all_functions__`

``` SQL
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA {schema} TO {role}
```


<a name="insert-on-all-tables"></a>
### ACL `__insert_on_all_tables__`

``` SQL
GRANT INSERT ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="references-on-all-tables"></a>
### ACL `__references_on_all_tables__`

``` SQL
GRANT REFERENCES ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="select-on-all-sequences"></a>
### ACL `__select_on_all_sequences__`

``` SQL
GRANT SELECT ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```


<a name="select-on-all-tables"></a>
### ACL `__select_on_all_tables__`

``` SQL
GRANT SELECT ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="temporary"></a>
### ACL `__temporary__`

``` SQL
GRANT TEMPORARY ON DATABASE {database} TO {role};
```


<a name="trigger-on-all-tables"></a>
### ACL `__trigger_on_all_tables__`

``` SQL
GRANT TRIGGER ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="truncate-on-all-tables"></a>
### ACL `__truncate_on_all_tables__`

``` SQL
GRANT TRUNCATE ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="update-on-all-sequences"></a>
### ACL `__update_on_all_sequences__`

``` SQL
GRANT UPDATE ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```


<a name="update-on-all-tables"></a>
### ACL `__update_on_all_tables__`

``` SQL
GRANT UPDATE ON ALL TABLES IN SCHEMA {schema} TO {role}
```


<a name="usage-on-all-sequences"></a>
### ACL `__usage_on_all_sequences__`

``` SQL
GRANT USAGE ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```


<a name="usage-on-schemas"></a>
### ACL `__usage_on_schemas__`

``` SQL
GRANT USAGE ON SCHEMA {schema} TO {role};
```


<a name="usage-on-types"></a>
### ACL `__usage_on_types__`

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT USAGE ON TYPES TO {role};
```

