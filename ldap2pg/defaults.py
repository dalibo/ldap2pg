_datacl_tpl = dict(
    type='datacl',
    inspect="""\
    WITH d AS (
        SELECT
            (aclexplode(datacl)).grantee AS grantee,
            (aclexplode(datacl)).privilege_type AS priv
        FROM pg_catalog.pg_database
        WHERE datname = current_database()
    )
    SELECT NULL as namespace, r.rolname
    FROM pg_catalog.pg_roles AS r
    JOIN d ON d.grantee = r.oid AND d.priv = '%(privilege)s'
    """.replace(' ' * 4, ''),
    grant="GRANT %(privilege)s ON DATABASE {database} TO {role};",
    revoke="REVOKE %(privilege)s ON DATABASE {database} FROM {role};",

)

_nspacl_tpl = dict(
    type="nspacl",
    inspect="""\
    WITH n AS (
      SELECT
        n.nspname AS namespace,
        (aclexplode(nspacl)).grantee AS grantee,
        (aclexplode(nspacl)).privilege_type AS priv
      FROM pg_catalog.pg_namespace AS n
    )
    SELECT
      n.namespace,
      r.rolname
    FROM pg_catalog.pg_roles AS r
    JOIN n ON n.grantee = r.oid AND n.priv = '%(privilege)s'
    ORDER BY 1, 2;
    """.replace(' ' * 4, ''),
    grant="GRANT %(privilege)s ON SCHEMA {schema} TO {role};",
    revoke="REVOKE %(privilege)s ON SCHEMA {schema} FROM {role};",
)


_tblacl_tpl = dict(
    type='nspacl',
    inspect="""\
    WITH
    namespace_tables AS (
      -- All namespaces and available relations aggregated.
      SELECT
        nsp.oid,
        nsp.nspname,
        array_agg(rel.relname ORDER BY rel.relname) AS tables
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_class rel
        ON rel.relnamespace = nsp.oid AND relkind = 'r'
      WHERE nspname NOT LIKE 'pg_%%'
      GROUP BY 1, 2
    ),
    all_grants AS (
      -- All grants on tables, aggregated by relname
      SELECT
        relnamespace,
        (aclexplode(relacl)).privilege_type,
        (aclexplode(relacl)).grantee,
        array_agg(relname ORDER BY relname) AS tables
      FROM pg_catalog.pg_class
      WHERE relkind = 'r'
      GROUP BY 1, 2, 3
    )
    SELECT
      nspname,
      rolname,
      nsp.tables = rels.tables AS "full"
    FROM namespace_tables AS nsp
    CROSS JOIN pg_catalog.pg_roles AS rol
    LEFT OUTER JOIN all_grants AS rels
      ON relnamespace = nsp.oid
         AND grantee = rol.oid
         AND privilege_type = '%(privilege)s'
    ORDER BY 1, 2
    """.replace('\n    ', '\n'),
    grant="GRANT %(privilege)s ON ALL TABLES IN SCHEMA {schema} TO {role}",
    revoke="REVOKE %(privilege)s ON ALL TABLES IN SCHEMA {schema} FROM {role}",
)


_defacl_tpl = dict(
    type="defacl",
    inspect="""\
    SELECT
      nspname,
      pg_catalog.pg_get_userbyid((aclexplode(defaclacl)).grantee) AS grantee,
      TRUE AS full,
      pg_catalog.pg_get_userbyid(defaclrole) AS owner
    FROM pg_catalog.pg_default_acl
    JOIN pg_catalog.pg_namespace nsp ON nsp.oid = defaclnamespace
    WHERE defaclobjtype = '%(t)s'
    ORDER BY 1, 2, 4;
    """.replace(' ' * 4, ''),
    grant="""\
    ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
    GRANT %(privilege)s ON %(TYPE)s TO {role};
    """.replace(' ' * 4, ''),
    revoke="""\
    ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
    REVOKE %(privilege)s ON %(TYPE)s FROM {role};
    """.replace(' ' * 4, ''),
)


_types = {
    'f': 'FUNCTIONS',
    'r': 'TABLES',
    't': 'TYPES',
    'S': 'SEQUENCES',
}


def make_acl(tpl, name, t, privilege):
    return name, dict(
        (k, v % (dict(t=t, TYPE=_types.get(t), privilege=privilege)))
        for k, v in tpl.items()
    )


def make_well_known_acls():
    acls = dict([
        make_acl(_datacl_tpl, '__connect__', None, 'CONNECT'),
        make_acl(_nspacl_tpl, '__usage_on_schema__', None, 'USAGE'),
        make_acl(_defacl_tpl, '__delete__', 't', 'DELETE'),
        make_acl(_defacl_tpl, '__execute__', 'f', 'EXECUTE'),
        make_acl(_defacl_tpl, '__insert__', 't', 'INSERT'),
        make_acl(_defacl_tpl, '__references__', 'r', 'REFERENCES'),
        make_acl(_defacl_tpl, '__truncate__', 'r', 'TRUNCATE'),
        make_acl(_defacl_tpl, '__default_select_on_tables__', 'r', 'SELECT'),
        make_acl(_defacl_tpl, '__select_on_sequences__', 'S', 'SELECT'),
        make_acl(_defacl_tpl, '__usage_on_types__', 't', 'USAGE'),
        make_acl(_defacl_tpl, '__update_on_sequences__', 'S', 'UPDATE'),
        make_acl(_defacl_tpl, '__update_on_tables__', 'r', 'UPDATE'),
        make_acl(_tblacl_tpl, '__select_on_all_tables__', None, 'SELECT'),
    ])

    acls.update(dict(
        __select_on_tables__=[
            '__default_select_on_tables__',
            '__select_on_all_tables__',
        ],
    ))

    return acls
