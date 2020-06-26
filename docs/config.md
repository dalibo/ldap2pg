<!-- markdownlint-disable MD033 MD041 MD046 -->

<h1><tt>ldap2pg.yml</tt></h1>

`ldap2pg` accepts a YAML configuration file usually named `ldap2pg.yml` and put
in working directory. Everything can be configured from the YAML file:
verbosity, LDAP and Postgres credentials, LDAP queries, privileges and
mappings.

!!! warning

    `ldap2pg` **requires** a config file where the synchronization map
    is described.


## File Location

`ldap2pg` searches for files in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set, `ldap2pg` skip searching the standard
file locations. You can specify `-` to read configuration from standard input.
This is helpful to feed `ldap2pg` with dynamic configuration.


## File Structure & Example

`ldap2pg.yml` is split in several sections :

- `postgres` : setup Postgres connexion and inspection queries.
- `ldap` : setup LDAP connexion.
- `privileges` : the definition of privileges.
- `sync_map` : the list of LDAP queries and associated mapping to roles and
  grants.
- finally some global parameters (verbosity, etc.).

We provide a simple well commented
[ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml), tested
on CI. If you don't know how to begin, it can be a good starting point.

!!! note

    If you have trouble finding the right configuration for your needs, feel free to
    [file an issue](https://github.com/dalibo/ldap2pg/issues/new) to get help.


## About YAML

YAML is a super-set of JSON. A JSON document is a valid YAML document. YAML very
permissive format where indentation is meaningful. See [this YAML
cheatsheet](https://medium.com/@kenichishibata/yaml-to-json-cheatsheet-c3ac3ef519b8)
for some example.


## Postgres Parameters

The `postgres` section defines connection parameters and queries for Postgres.

``` yaml
postgres:
  dsn: postgres://user@%2Fvar%2Frun%2Fpostgresql:port/
```

!!! warning

    `ldap2pg` refuses to read a password from a group readable or world
    readable `ldap2pg.yml`.


## LDAP Parameters

``` yaml
ldap:
  uri: ldap://ldap2pg.local:389
  binddn: cn=admin,dc=ldap2pg,dc=local
  user: saslusername
  password: SECRET
```


## `sync_map`

The synchronization map is a YAML list. We call each item a *mapping*. Three
sections compose a mapping:

- A `description` entry with a string logged before this mapping is processed.
- A `ldap` section describing a LDAP query.
- A `role` or `roles` section describing on or more rules to create [Postgres
  role](https://www.postgresql.org/docs/current/static/user-manag.html) from
  LDAP entries.
- A `grant` section describing on or more grant from LDAP entries.

`ldap` entry is optional, however either one of `roles` or `grant` is required.

!!! tip

    Defining the right sync map can be tedious. Start with is simple
    sync map to setup Postgres and LDAP connexion first and then define detailed
    synchronisation steps. Here is the simplest sync map:

    <pre class="highlight"><code class="language-yaml">sync_map:
    - role: toto
    </code></pre>

    It just means you want a role named `toto` in the cluster.


## Various Parameters

Finally, `ldap2pg.yml` contains various plain parameters for `ldap2pg`
behaviour.

``` yaml
# Colorization. env var: COLOR=<anything>
color: yes

# Verbose messages. Includes SQL and LDAP queries. env var: VERBOSITY
verbosity: 5

# Dry mode. env var: DRY=<anything>
dry: yes
```


## Shortcuts

If the file is a YAML list, `ldap2pg` puts the list as `sync_map`. The two
following configurations are strictly equivalent:

``` console
$ ldap2pg -c -
- admin
$ ldap2pg -c -
sync_map:
- roles:
  - names:
    - admin
$
```

`database`, `schema`, `role`, `name`, `parent` and `member` can be either a
string or a list of strings. These keys have plural aliases, respectively
`databases`, `schema`, `roles`, `names`, `parents` and `members`.

<!-- Local Variables: -->
<!-- ispell-dictionary: "american" -->
<!-- End: -->
