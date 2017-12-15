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


# ALL TABLES is tricky because we have to manage partial grant. But the
# trickiest comes when there is no tables in a namespace. In this case, is it
# granted or revoked ? We have to tell ldap2pg that this ACL is irrelevant on
# this schema.
#
# Here is a truth table:
#
#  FOR GRANT | no grant | partial grant | fully granted
# -----------+----------+---------------+---------------
#  no tables |   NOOP   |      N/D      |      N/D
# -----------+----------+---------------+---------------
#  1+ tables |   GRANT  |     GRANT     |      NOOP
# -----------+----------+---------------+---------------
#
# FOR REVOKE | no grant | partial grant | fully granted
# -----------+----------+---------------+---------------
#  no tables |   NOOP   |      N/D      |      N/D
# -----------+----------+---------------+---------------
#  1+ tables |   NOOP   |     REVOKE    |     REVOKE
# -----------+----------+---------------+---------------
#
# When namespace has NO tables, we always return a row with full as NULL,
# meaning ACL is irrelevant : it is both granted and revoked.
#
# When namespace has tables, we compare grants to availables tables to
# determine if ACL is fully granted. If the ACL is not granted at all, we drop
# the row in WHERE clause to ensure the ACL is considered as revoked.
#
_tblacl_tpl = dict(
    type='nspacl',
    inspect="""\
    WITH
    namespace_tables AS (
      SELECT
        nsp.oid,
        nsp.nspname,
        array_agg(rel.relname ORDER BY rel.relname)
          FILTER (WHERE rel.relname IS NOT NULL) AS tables
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_class rel
        ON rel.relnamespace = nsp.oid AND relkind = 'r'
      WHERE nspname NOT LIKE 'pg_%%'
      GROUP BY 1, 2
    ),
    all_grants AS (
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
      CASE
        WHEN nsp.tables IS NULL THEN NULL
        ELSE nsp.tables = COALESCE(grants.tables, ARRAY[]::name[])
      END AS "full"
    FROM namespace_tables AS nsp
    CROSS JOIN pg_catalog.pg_roles AS rol
    LEFT OUTER JOIN all_grants AS grants
      ON relnamespace = nsp.oid
         AND grantee = rol.oid
         AND privilege_type = '%(privilege)s'
    WHERE NOT (nsp.tables IS NOT NULL AND grants.tables IS NULL)
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


def make_table_acls(privilege, namefmt='__%s__'):
    fmtargs = (privilege.lower(),)
    all_ = '__%s_all__' % fmtargs
    default = '__%s_default__' % fmtargs
    name = namefmt % fmtargs
    return dict([
        make_acl(_tblacl_tpl, all_, 'r', privilege.upper()),
        make_acl(_defacl_tpl, default, 'r', privilege.upper()),
        (name, [all_, default]),
    ])


def make_well_known_acls():
    acls = dict([
        make_acl(_datacl_tpl, '__connect__', None, 'CONNECT'),
        make_acl(_nspacl_tpl, '__usage_on_schema__', None, 'USAGE'),
        make_acl(_defacl_tpl, '__execute__', 'f', 'EXECUTE'),
        make_acl(_defacl_tpl, '__select_on_sequences__', 'S', 'SELECT'),
        make_acl(_defacl_tpl, '__usage_on_types__', 't', 'USAGE'),
        make_acl(_defacl_tpl, '__update_on_sequences__', 'S', 'UPDATE'),
    ])

    for privilege in 'DELETE', 'INSERT', 'REFERENCES', 'TRUNCATE':
        acls.update(make_table_acls(privilege))
    for privilege in 'SELECT', 'UPDATE':
        acls.update(make_table_acls(privilege, '__%s_on_tables__'))

    return acls
