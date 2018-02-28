from textwrap import dedent

_datacl_tpl = dict(
    type='datacl',
    inspect=dedent("""\
    WITH grants AS (
      SELECT
        (aclexplode(datacl)).grantee AS grantee,
        (aclexplode(datacl)).privilege_type AS priv
      FROM pg_catalog.pg_database
      WHERE datname = current_database()
      UNION
      SELECT q.*
      FROM (VALUES (0, 'CONNECT'), (0, 'TEMPORARY')) AS q
      CROSS JOIN pg_catalog.pg_database
      WHERE datacl IS NULL AND datname = current_database()
    )
    SELECT
      NULL as namespace,
      COALESCE(rolname, 'public')
    FROM grants
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE (grantee = 0 OR rolname IS NOT NULL)
      AND grants.priv = '%(privilege)s';
    """),
    grant="GRANT %(privilege)s ON DATABASE {database} TO {role};",
    revoke="REVOKE %(privilege)s ON DATABASE {database} FROM {role};",

)

_global_defacl_tpl = dict(
    type='globaldefacl',
    inspect=dedent("""\
    WITH
    grants AS (
      SELECT
        defaclrole AS owner,
        (aclexplode(defaclacl)).grantee,
        (aclexplode(defaclacl)).privilege_type AS priv
      FROM pg_default_acl AS def
      WHERE defaclnamespace = 0
      UNION
      SELECT
        rol.oid AS owner,
        0 AS grantee,
        'EXECUTE' AS priv
      FROM pg_roles AS rol
      LEFT OUTER JOIN pg_catalog.pg_default_acl AS defacl
        ON defacl.defaclrole = rol.oid AND defacl.defaclnamespace = 0
      WHERE defaclacl IS NULL
    )
    SELECT
      NULL AS "schema",
      COALESCE(rolname, 'public') as rolname,
      TRUE AS "full",
      pg_catalog.pg_get_userbyid(owner) AS owner
    FROM grants
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE (rolname IS NOT NULL OR grantee = 0)
      AND priv = '%(privilege)s'
    """),
    grant=(
        "ALTER DEFAULT PRIVILEGES FOR ROLE {owner}"
        " GRANT %(privilege)s ON %(TYPE)s TO {role};"),
    revoke=(
        "ALTER DEFAULT PRIVILEGES FOR ROLE {owner}"
        " REVOKE %(privilege)s ON %(TYPE)s FROM {role};"),
)

_defacl_tpl = dict(
    type="defacl",
    inspect=dedent("""\
    WITH
    grants AS (
      SELECT
        defaclnamespace,
        defaclrole,
        (aclexplode(defaclacl)).grantee AS grantee,
        (aclexplode(defaclacl)).privilege_type AS priv
      FROM pg_catalog.pg_default_acl
      WHERE defaclobjtype IN %(t)s
    )
    SELECT
      nspname,
      COALESCE(rolname, 'public') AS rolname,
      TRUE AS full,
      pg_catalog.pg_get_userbyid(defaclrole) AS owner
    FROM grants
    JOIN pg_catalog.pg_namespace nsp ON nsp.oid = defaclnamespace
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE (grantee = 0 OR rolname IS NOT NULL)
      AND priv = '%(privilege)s'
    ORDER BY 1, 2, 4;
    """),
    grant=dedent("""\
    ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
    GRANT %(privilege)s ON %(TYPE)s TO {role};
    """),
    revoke=dedent("""\
    ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema}
    REVOKE %(privilege)s ON %(TYPE)s FROM {role};
    """),
)

_nspacl_tpl = dict(
    type="nspacl",
    inspect=dedent("""\
    WITH grants AS (
      SELECT
        nspname,
        (aclexplode(nspacl)).grantee AS grantee,
        (aclexplode(nspacl)).privilege_type AS priv
      FROM pg_catalog.pg_namespace
    )
    SELECT
      nspname,
      COALESCE(rolname, 'public') AS rolname
    FROM grants
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE (grantee = 0 OR rolname IS NOT NULL)
      AND grants.priv = '%(privilege)s'
    ORDER BY 1, 2;
    """),
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
    inspect=dedent("""\
    WITH
    namespace_rels AS (
      SELECT
        nsp.oid,
        nsp.nspname,
        array_agg(rel.relname ORDER BY rel.relname)
          FILTER (WHERE rel.relname IS NOT NULL) AS rels
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_class AS rel
        ON rel.relnamespace = nsp.oid AND relkind IN %(t)s
      GROUP BY 1, 2
    ),
    all_grants AS (
      SELECT
        relnamespace,
        (aclexplode(relacl)).privilege_type,
        (aclexplode(relacl)).grantee,
        array_agg(relname ORDER BY relname) AS rels
      FROM pg_catalog.pg_class
      WHERE relkind IN %(t)s
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
    """),
    grant="GRANT %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} TO {role}",
    revoke=(
        "REVOKE %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} FROM {role}"),
)


_allprocacl_tpl = dict(
    type='nspacl',
    inspect=dedent("""\
    WITH
    grants AS (SELECT
      pronamespace, grantee, priv,
      array_agg(DISTINCT proname ORDER BY proname) AS procs
      FROM (
        SELECT
          pronamespace,
          proname,
          (aclexplode(proacl)).grantee,
          (aclexplode(proacl)).privilege_type AS priv
        FROM pg_catalog.pg_proc
        UNION
        SELECT
          pronamespace, proname,
          0 AS grantee,
          'EXECUTE' AS priv
        FROM pg_catalog.pg_proc
        WHERE proacl IS NULL
      ) AS grants
      GROUP BY 1, 2, 3
    ),
    namespaces AS (
      SELECT
        nsp.oid, nsp.nspname,
        array_agg(DISTINCT pro.proname ORDER BY pro.proname)
          FILTER (WHERE pro.proname IS NOT NULL) AS procs
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_proc AS pro
        ON pro.pronamespace = nsp.oid
      GROUP BY 1, 2
    ),
    roles AS (
      SELECT oid, rolname
      FROM pg_catalog.pg_roles
      UNION
      SELECT 0, 'public'
    )
    SELECT
      nspname, rolname,
      CASE
        WHEN nsp.procs IS NULL THEN NULL
        ELSE nsp.procs = COALESCE(grants.procs, ARRAY[]::name[])
      END AS "full"
    FROM namespaces AS nsp
    CROSS JOIN roles
    LEFT OUTER JOIN grants
      ON pronamespace = nsp.oid AND grants.grantee = roles.oid
    WHERE NOT (nsp.procs IS NOT NULL AND grants.procs IS NULL)
      AND (priv IS NULL OR priv = '%(privilege)s')
    ORDER BY 1, 2;
    """),
    grant="GRANT %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} TO {role}",
    revoke=(
        "REVOKE %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} FROM {role}"),
)


_types = {
    'FUNCTIONS': ('f',),
    'TABLES': ('r', 'v'),
    'TYPES': ('T',),
    'SEQUENCES': ('S',),
}


def make_acl(tpl, name, TYPE, privilege):
    t = _types.get(TYPE)
    if t:
        # Loose SQL formatting
        t = '(%s)' % (', '.join(['%r' % i for i in t]))
    return name, dict(
        (k, v % (dict(
            t=t,
            TYPE=TYPE,
            privilege=privilege.upper(),
        )))
        for k, v in tpl.items()
    )


def make_proc_acls(privilege, TYPE='FUNCTIONS',
                   namefmt='__%(privilege)s_on_%(type)s__'):
    fmtkw = dict(privilege=privilege.lower(), type=TYPE.lower())
    all_ = '__%(privilege)s_on_all_%(type)s__' % fmtkw
    default = '__default_%(privilege)s_on_%(type)s__' % fmtkw
    global_def = '__global_default_%(privilege)s_on_%(type)s__' % fmtkw
    name = namefmt % fmtkw
    return dict([
        make_acl(_allprocacl_tpl, all_, TYPE, privilege),
        make_acl(_defacl_tpl, default, TYPE, privilege),
        make_acl(_global_defacl_tpl, global_def, TYPE, privilege),
        (name, [all_, default, global_def]),
    ])


def make_rel_acls(privilege, TYPE, namefmt='__%(privilege)s_on_%(type)s__'):
    fmtkw = dict(privilege=privilege.lower(), type=TYPE.lower())
    all_ = '__%(privilege)s_on_all_%(type)s__' % fmtkw
    default = '__default_%(privilege)s_on_%(type)s__' % fmtkw
    name = namefmt % fmtkw
    return dict([
        make_acl(_allrelacl_tpl, all_, TYPE, privilege),
        make_acl(_defacl_tpl, default, TYPE, privilege),
        (name, [all_, default]),
    ])


def make_well_known_acls():
    acls = dict([
        make_acl(_datacl_tpl, '__connect__', None, 'CONNECT'),
        make_acl(_datacl_tpl, '__temporary__', None, 'TEMPORARY'),
        make_acl(_nspacl_tpl, '__create_on_schemas__', None, 'CREATE'),
        make_acl(_nspacl_tpl, '__usage_on_schemas__', None, 'USAGE'),
        make_acl(_nspacl_tpl, '__usage_on_types__', 'TYPES', 'USAGE'),
    ])

    acls.update(make_proc_acls('EXECUTE', 'FUNCTIONS'))
    acls['__execute__'] = ['__execute_on_functions__']

    for privilege in 'DELETE', 'INSERT', 'REFERENCES', 'TRIGGER', 'TRUNCATE':
        acls.update(
            make_rel_acls(privilege, 'TABLES'))
        alias = '__%s__' % (privilege.lower(),)
        acls[alias] = ['__%s_on_tables__' % (privilege.lower(),)]

    for privilege in 'SELECT', 'UPDATE':
        acls.update(make_rel_acls(privilege, 'TABLES'))
        acls.update(make_rel_acls(privilege, 'SEQUENCES'))

    acls.update(make_rel_acls('USAGE', 'SEQUENCES'))

    acls['__all_on_schemas__'] = [
        '__create_on_schemas__',
        '__usage_on_schemas__',
    ]

    acls['__all_on_sequences__'] = [
        '__select_on_sequences__',
        '__update_on_sequences__',
        '__usage_on_sequences__',
    ]

    acls['__all_on_tables__'] = [
        '__delete__',
        '__insert__',
        '__references__',
        '__select_on_tables__',
        '__trigger__',
        '__truncate__',
        '__update_on_tables__',
    ]

    return acls
