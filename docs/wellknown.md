<!-- GENERATED FROM docs/wellknown.md.j2 -->
<!--*- markdown -*-->

<h1>Well-known Privileges</h1>

ldap2pg provides some well-known privileges for recurrent usage. There is **no
warranty** on these privileges. You have to check privileges configuration on
your databases just like you should do with your own code.

The true added-value of well-known privileges is the `inspect` queries
associated and the boilerplate saved for declaring all `GRANT` queries.


## Using Well-known Privileges

Well-known privilege starts and ends with `__`. ldap2pg [disables
privileges](privileges.md#enabling-privilege) starting with `_`. Thus you have
to include well-known privileges in a group to enable them. If two groups
reference the same privilege, it will be deduplicated, don't worry.

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
    schema: __all__
    role: admins
```

Well-known privilege name follows the following loose convention:

- `..._on_all_tables__` is equivalent to `GRANT ... ON ALL TABLES IN SCHEMA ...`.
- `__default_...__` is equivalent to `ALTER DEFAULT PRIVILEGES ... IN SCHEMA ...`.
- `__..._on_tables__` gathers `__..._on_all_tables__` and
  `__default_..._on_tables__`.
- Group starting with `__all_on_...__` is *equivalent* to `ALL PRIVILEGES` in
  SQL.
- A privilege specific to one object type does not have `_on_<type>__` e.g.
  `__delete_on_tables__` is aliased to `__delete__`.

This page does not document the SQL standard and the meaning of each SQL
privileges. You will find the documentation of SQL privileges in [Postgresql
GRANT documentation](https://www.postgresql.org/docs/current/sql-grant.html) and
[ALTER DEFAULT PRIVILEGES
documentation](https://www.postgresql.org/docs/current/sql-alterdefaultprivileges.html).


## Privilege Groups

Next is an extensive, boring, list of all well known privilege groups in
`master`. Each group is documented by its name and the list of included
privilege. Each privilege name point the the detail of privilege definition.

Actually, a group like `__all_on_tables__` is implemented as group of groups.
But for the sake of simplicity, the documentation lists the constructed list
of concrete privileges finally included.

Here we go.


### Group `__all_on_schemas__`  { #all-on-schemas data-toc-label="&#95;&#95;all&#95;on&#95;schemas&#95;&#95;" }

Includes:

- [`__create_on_schemas__`](#create-on-schemas)
- [`__usage_on_schemas__`](#usage-on-schemas)



### Group `__all_on_sequences__`  { #all-on-sequences data-toc-label="&#95;&#95;all&#95;on&#95;sequences&#95;&#95;" }

Includes:

- [`__default_select_on_sequences__`](#default-select-on-sequences)
- [`__default_update_on_sequences__`](#default-update-on-sequences)
- [`__default_usage_on_sequences__`](#default-usage-on-sequences)
- [`__select_on_all_sequences__`](#select-on-all-sequences)
- [`__update_on_all_sequences__`](#update-on-all-sequences)
- [`__usage_on_all_sequences__`](#usage-on-all-sequences)



### Group `__all_on_tables__`  { #all-on-tables data-toc-label="&#95;&#95;all&#95;on&#95;tables&#95;&#95;" }

Includes:

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



### Group `__delete_on_tables__`  { #delete-on-tables data-toc-label="&#95;&#95;delete&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_delete_on_tables__`](#default-delete-on-tables)
- [`__delete_on_all_tables__`](#delete-on-all-tables)

Alias: `__delete__`

### Group `__execute_on_functions__`  { #execute-on-functions data-toc-label="&#95;&#95;execute&#95;on&#95;functions&#95;&#95;" }

Includes:

- [`__default_execute_on_functions__`](#default-execute-on-functions)
- [`__execute_on_all_functions__`](#execute-on-all-functions)
- [`__global_default_execute_on_functions__`](#global-default-execute-on-functions)

Alias: `__execute__`

### Group `__insert_on_tables__`  { #insert-on-tables data-toc-label="&#95;&#95;insert&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_insert_on_tables__`](#default-insert-on-tables)
- [`__insert_on_all_tables__`](#insert-on-all-tables)

Alias: `__insert__`

### Group `__references_on_tables__`  { #references-on-tables data-toc-label="&#95;&#95;references&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_references_on_tables__`](#default-references-on-tables)
- [`__references_on_all_tables__`](#references-on-all-tables)

Alias: `__references__`

### Group `__select_on_sequences__`  { #select-on-sequences data-toc-label="&#95;&#95;select&#95;on&#95;sequences&#95;&#95;" }

Includes:

- [`__default_select_on_sequences__`](#default-select-on-sequences)
- [`__select_on_all_sequences__`](#select-on-all-sequences)



### Group `__select_on_tables__`  { #select-on-tables data-toc-label="&#95;&#95;select&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_select_on_tables__`](#default-select-on-tables)
- [`__select_on_all_tables__`](#select-on-all-tables)



### Group `__trigger_on_tables__`  { #trigger-on-tables data-toc-label="&#95;&#95;trigger&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_trigger_on_tables__`](#default-trigger-on-tables)
- [`__trigger_on_all_tables__`](#trigger-on-all-tables)

Alias: `__trigger__`

### Group `__truncate_on_tables__`  { #truncate-on-tables data-toc-label="&#95;&#95;truncate&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_truncate_on_tables__`](#default-truncate-on-tables)
- [`__truncate_on_all_tables__`](#truncate-on-all-tables)

Alias: `__truncate__`

### Group `__update_on_sequences__`  { #update-on-sequences data-toc-label="&#95;&#95;update&#95;on&#95;sequences&#95;&#95;" }

Includes:

- [`__default_update_on_sequences__`](#default-update-on-sequences)
- [`__update_on_all_sequences__`](#update-on-all-sequences)



### Group `__update_on_tables__`  { #update-on-tables data-toc-label="&#95;&#95;update&#95;on&#95;tables&#95;&#95;" }

Includes:

- [`__default_update_on_tables__`](#default-update-on-tables)
- [`__update_on_all_tables__`](#update-on-all-tables)



### Group `__usage_on_sequences__`  { #usage-on-sequences data-toc-label="&#95;&#95;usage&#95;on&#95;sequences&#95;&#95;" }

Includes:

- [`__default_usage_on_sequences__`](#default-usage-on-sequences)
- [`__usage_on_all_sequences__`](#usage-on-all-sequences)



## Single Privileges

Next is the list of well-known privileges. Each is associated with a `REVOKE`
query and an `inspect` query implementing full inspection of grantees,
including built-in grants to PUBLIC.

For the actual meaning of each SQL privileges, refer to official [PostgreSQL
documentation of
`GRANT`](https://www.postgresql.org/docs/current/static/sql-grant.html)
statement.



### Privilege `__connect__`  { #connect data-toc-label="&#95;&#95;connect&#95;&#95;" }

``` SQL
GRANT CONNECT ON DATABASE {database} TO {role};
```



### Privilege `__create_on_schemas__`  { #create-on-schemas data-toc-label="&#95;&#95;create&#95;on&#95;schemas&#95;&#95;" }

``` SQL
GRANT CREATE ON SCHEMA {schema} TO {role};
```



### Privilege `__default_delete_on_tables__`  { #default-delete-on-tables data-toc-label="&#95;&#95;default&#95;delete&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT DELETE ON TABLES TO {role};
```



### Privilege `__default_execute_on_functions__`  { #default-execute-on-functions data-toc-label="&#95;&#95;default&#95;execute&#95;on&#95;functions&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT EXECUTE ON FUNCTIONS TO {role};
```



### Privilege `__default_insert_on_tables__`  { #default-insert-on-tables data-toc-label="&#95;&#95;default&#95;insert&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT INSERT ON TABLES TO {role};
```



### Privilege `__default_references_on_tables__`  { #default-references-on-tables data-toc-label="&#95;&#95;default&#95;references&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT REFERENCES ON TABLES TO {role};
```



### Privilege `__default_select_on_sequences__`  { #default-select-on-sequences data-toc-label="&#95;&#95;default&#95;select&#95;on&#95;sequences&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT SELECT ON SEQUENCES TO {role};
```



### Privilege `__default_select_on_tables__`  { #default-select-on-tables data-toc-label="&#95;&#95;default&#95;select&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT SELECT ON TABLES TO {role};
```



### Privilege `__default_trigger_on_tables__`  { #default-trigger-on-tables data-toc-label="&#95;&#95;default&#95;trigger&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT TRIGGER ON TABLES TO {role};
```



### Privilege `__default_truncate_on_tables__`  { #default-truncate-on-tables data-toc-label="&#95;&#95;default&#95;truncate&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT TRUNCATE ON TABLES TO {role};
```



### Privilege `__default_update_on_sequences__`  { #default-update-on-sequences data-toc-label="&#95;&#95;default&#95;update&#95;on&#95;sequences&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT UPDATE ON SEQUENCES TO {role};
```



### Privilege `__default_update_on_tables__`  { #default-update-on-tables data-toc-label="&#95;&#95;default&#95;update&#95;on&#95;tables&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT UPDATE ON TABLES TO {role};
```



### Privilege `__default_usage_on_sequences__`  { #default-usage-on-sequences data-toc-label="&#95;&#95;default&#95;usage&#95;on&#95;sequences&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT USAGE ON SEQUENCES TO {role};
```



### Privilege `__default_usage_on_types__`  { #default-usage-on-types data-toc-label="&#95;&#95;default&#95;usage&#95;on&#95;types&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
GRANT USAGE ON TYPES TO {role};
```

Alias: `__usage_on_types__`

### Privilege `__delete_on_all_tables__`  { #delete-on-all-tables data-toc-label="&#95;&#95;delete&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT DELETE ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__execute_on_all_functions__`  { #execute-on-all-functions data-toc-label="&#95;&#95;execute&#95;on&#95;all&#95;functions&#95;&#95;" }

``` SQL
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA {schema} TO {role}
```



### Privilege `__global_default_execute_on_functions__`  { #global-default-execute-on-functions data-toc-label="&#95;&#95;global&#95;default&#95;execute&#95;on&#95;functions&#95;&#95;" }

``` SQL
ALTER DEFAULT PRIVILEGES FOR ROLE {owner} GRANT EXECUTE ON FUNCTIONS TO {role};
```



### Privilege `__insert_on_all_tables__`  { #insert-on-all-tables data-toc-label="&#95;&#95;insert&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT INSERT ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__references_on_all_tables__`  { #references-on-all-tables data-toc-label="&#95;&#95;references&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT REFERENCES ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__select_on_all_sequences__`  { #select-on-all-sequences data-toc-label="&#95;&#95;select&#95;on&#95;all&#95;sequences&#95;&#95;" }

``` SQL
GRANT SELECT ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```



### Privilege `__select_on_all_tables__`  { #select-on-all-tables data-toc-label="&#95;&#95;select&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT SELECT ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__temporary__`  { #temporary data-toc-label="&#95;&#95;temporary&#95;&#95;" }

``` SQL
GRANT TEMPORARY ON DATABASE {database} TO {role};
```



### Privilege `__trigger_on_all_tables__`  { #trigger-on-all-tables data-toc-label="&#95;&#95;trigger&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT TRIGGER ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__truncate_on_all_tables__`  { #truncate-on-all-tables data-toc-label="&#95;&#95;truncate&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT TRUNCATE ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__update_on_all_sequences__`  { #update-on-all-sequences data-toc-label="&#95;&#95;update&#95;on&#95;all&#95;sequences&#95;&#95;" }

``` SQL
GRANT UPDATE ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```



### Privilege `__update_on_all_tables__`  { #update-on-all-tables data-toc-label="&#95;&#95;update&#95;on&#95;all&#95;tables&#95;&#95;" }

``` SQL
GRANT UPDATE ON ALL TABLES IN SCHEMA {schema} TO {role}
```



### Privilege `__usage_on_all_sequences__`  { #usage-on-all-sequences data-toc-label="&#95;&#95;usage&#95;on&#95;all&#95;sequences&#95;&#95;" }

``` SQL
GRANT USAGE ON ALL SEQUENCES IN SCHEMA {schema} TO {role}
```



### Privilege `__usage_on_schemas__`  { #usage-on-schemas data-toc-label="&#95;&#95;usage&#95;on&#95;schemas&#95;&#95;" }

``` SQL
GRANT USAGE ON SCHEMA {schema} TO {role};
```


