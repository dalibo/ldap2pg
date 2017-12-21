<!--*- markdown -*-->

<h1><tt>ldap2pg.yml</tt></h1>

`ldap2pg` accepts a YAML configuration file. Everything can be configured from
the YAML file: verbosity, real mode, LDAP and Postgres credentials, LDAP
queries, ACL and mappings.

!!! warning

    `ldap2pg` **requires** a config file where the synchronization map
    is described.


`ldap2pg.yml` is splitted in several sections, unordered :

- `postgres` : setup Postgres connexion and queries.
- `ldap` : setup LDAP connexion.
- `acls` : the definition of grants.
- `sync_map` : the list of LDAP queries and associated mapping to roles and
  grants.
- finally some global parameters (verbosity, etc.).

If the file is a YAML list, `ldap2pg` puts the list as `sync_map`. The two
following configurations are strictly equivalent:

``` console
$ echo '- role: admin' | ldap2pg -c -
...
$ ldap2pg -c -
sync_map:
- roles:
  - names:
    - admin
$
```


We provide a simple well commented
[ldap2pg.yml](https://github.com/dalibo/ldap2pg/blob/master/ldap2pg.yml), tested
on CI. If you don't know how to begin, it can be a goot starting point.

!!! note

    If you have trouble finding the right configuration for your needs, feel free to
    [file an issue](https://github.com/dalibo/ldap2pg/issues/new) to get help.


## Postgres parameters

The `postgres` section defines connection parameters and queries for Postgres.

``` yaml
postgres:
  dsn: postgres://user@%2Fvar%2Frun%2Fpostgresql:port/
```

!!! warning

    `ldap2pg` refuses to read a password from a group readable or world
    readable `ldap2pg.yml`.


## LDAP parameters

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


## Various parameters

Finally, `ldap2pg.yml` contains various plain parameters for `ldap2pg`
behaviour.

``` yaml
# Colorization. env var: COLOR=<anything>
color: yes

# Verbose messages. Includes SQL and LDAP queries. env var: VERBOSE
verbose: no

# Dry mode. env var: DRY=<anything>
dry: yes
```


## File location

`ldap2pg` searches for files in the following order :

1. `ldap2pg.yml` in current working directory.
2. `~/.config/ldap2pg.yml`.
3. `/etc/ldap2pg.yml`.

If `LDAP2PG_CONFIG` or `--config` is set, `ldap2pg` skip searching the standard
file locations. You can specify `-` to read configuration from standard input.
This is helpful to feed `ldap2pg` with dynamic configuration.
