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

_defacl_tpl = dict(
    type="defacl",
    inspect="""\
    WITH
    grants AS (
      SELECT
        defaclnamespace,
        defaclrole,
        (aclexplode(defaclacl)).grantee AS grantee,
        (aclexplode(defaclacl)).privilege_type
      FROM pg_catalog.pg_default_acl
      WHERE defaclobjtype = '%(t)s'
    )
    SELECT
      nspname,
      pg_catalog.pg_get_userbyid(grantee) AS grantee,
      TRUE AS full,
      pg_catalog.pg_get_userbyid(defaclrole) AS owner
    FROM grants
    JOIN pg_catalog.pg_namespace nsp ON nsp.oid = defaclnamespace
    WHERE privilege_type = '%(privilege)s'
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
_allrelacl_tpl = dict(
    type='nspacl',
    inspect="""WITH
    namespace_rels AS (
      SELECT
        nsp.oid,
        nsp.nspname,
        array_agg(rel.relname ORDER BY rel.relname)
          FILTER (WHERE rel.relname IS NOT NULL) AS rels
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_class AS rel
        ON rel.relnamespace = nsp.oid AND relkind = '%(t)s'
      WHERE nspname NOT LIKE 'pg_%%'
      GROUP BY 1, 2
    ),
    all_grants AS (
      SELECT
        relnamespace,
        (aclexplode(relacl)).privilege_type,
        (aclexplode(relacl)).grantee,
        array_agg(relname ORDER BY relname) AS rels
      FROM pg_catalog.pg_class
      WHERE relkind = '%(t)s'
      GROUP BY 1, 2, 3
    )
    SELECT
      nspname,
      rolname,
      CASE
        WHEN nsp.rels IS NULL THEN NULL
        ELSE nsp.rels = COALESCE(grants.rels, ARRAY[]::name[])
      END AS "full"
    FROM namespace_rels AS nsp
    CROSS JOIN pg_catalog.pg_roles AS rol
    LEFT OUTER JOIN all_grants AS grants
      ON relnamespace = nsp.oid
         AND grantee = rol.oid
         AND privilege_type = '%(privilege)s'
    WHERE NOT (nsp.rels IS NOT NULL AND grants.rels IS NULL)
    ORDER BY 1, 2
    """.replace('\n    ', '\n'),
    grant="GRANT %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} TO {role}",
    revoke=(
        "REVOKE %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} FROM {role}"),
)


_allprocacl_tpl = dict(
    type='nspacl',
    inspect="""WITH
    namespace_procs AS (
      SELECT
        nsp.oid,
        nsp.nspname,
        array_agg(pro.proname ORDER BY pro.proname)
          FILTER (WHERE pro.proname IS NOT NULL) AS procs
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_proc AS pro
        ON pro.pronamespace = nsp.oid
      WHERE nspname NOT LIKE 'pg_%%'
      GROUP BY 1, 2
    ),
    all_grants AS (
      SELECT
        pronamespace,
        (aclexplode(proacl)).privilege_type,
        (aclexplode(proacl)).grantee,
        array_agg(proname ORDER BY proname) AS procs
      FROM pg_catalog.pg_proc
      GROUP BY 1, 2, 3
    )
    SELECT
      nspname,
      rolname,
      CASE
        WHEN nsp.procs IS NULL THEN NULL
        ELSE nsp.procs = COALESCE(grants.procs, ARRAY[]::name[])
      END AS "full"
    FROM namespace_procs AS nsp
    CROSS JOIN pg_catalog.pg_roles AS rol
    LEFT OUTER JOIN all_grants AS grants
      ON pronamespace = nsp.oid
         AND grantee = rol.oid
         AND privilege_type = '%(privilege)s'
    WHERE NOT (nsp.procs IS NOT NULL AND grants.procs IS NULL)
    ORDER BY 1, 2
    """.replace('\n    ', '\n'),
    grant="GRANT %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} TO {role}",
    revoke=(
        "REVOKE %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} FROM {role}"),
)


_types = {
    'f': 'FUNCTIONS',
    'r': 'TABLES',
    'T': 'TYPES',
    'S': 'SEQUENCES',
}


def make_acl(tpl, name, t, privilege):
    return name, dict(
        (k, v % (dict(t=t, TYPE=_types.get(t), privilege=privilege.upper())))
        for k, v in tpl.items()
    )


def make_proc_acls(privilege, t='f', namefmt='__%(privilege)s_on_%(type)s__'):
    fmtkw = dict(privilege=privilege.lower(), type=_types[t].lower())
    all_ = '__%(privilege)s_on_all_%(type)s__' % fmtkw
    default = '__default_%(privilege)s_on_%(type)s__' % fmtkw
    name = namefmt % fmtkw
    return dict([
        make_acl(_allprocacl_tpl, all_, t, privilege),
        make_acl(_defacl_tpl, default, t, privilege),
        (name, [all_, default]),
    ])


def make_rel_acls(privilege, t, namefmt='__%(privilege)s_on_%(type)s__'):
    fmtkw = dict(privilege=privilege.lower(), type=_types[t].lower())
    all_ = '__%(privilege)s_on_all_%(type)s__' % fmtkw
    default = '__default_%(privilege)s_on_%(type)s__' % fmtkw
    name = namefmt % fmtkw
    return dict([
        make_acl(_allrelacl_tpl, all_, t, privilege),
        make_acl(_defacl_tpl, default, t, privilege),
        (name, [all_, default]),
    ])


def make_well_known_acls():
    acls = dict([
        make_acl(_datacl_tpl, '__connect__', None, 'CONNECT'),
        make_acl(_nspacl_tpl, '__usage_on_schema__', None, 'USAGE'),
        make_acl(_defacl_tpl, '__usage_on_types__', 'T', 'USAGE'),
    ])

    acls.update(make_proc_acls('EXECUTE', 'f', namefmt='__%(privilege)s__'))

    for privilege in 'DELETE', 'INSERT', 'REFERENCES', 'TRUNCATE':
        acls.update(make_rel_acls(privilege, 'r', namefmt='__%(privilege)s__'))

    for privilege in 'SELECT', 'UPDATE':
        acls.update(make_rel_acls(privilege, 'r'))
        acls.update(make_rel_acls(privilege, 'S'))

    return acls
