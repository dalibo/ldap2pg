from itertools import chain
from textwrap import dedent

from .utils import string_types


shared_queries = dict(
    datacl=dedent("""\
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
      grants.priv AS key,
      NULL as namespace,
      COALESCE(rolname, 'public')
    FROM grants
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE grantee = 0 OR rolname IS NOT NULL
    """),
    defacl=dedent("""\
    WITH
    grants AS (
      SELECT
        defaclnamespace,
        defaclrole,
        (aclexplode(defaclacl)).grantee AS grantee,
        (aclexplode(defaclacl)).privilege_type AS priv,
        defaclobjtype AS objtype
      FROM pg_catalog.pg_default_acl
    )
    SELECT
      priv || '_on_' || objtype AS key,
      nspname,
      COALESCE(rolname, 'public') AS rolname,
      TRUE AS full,
      pg_catalog.pg_get_userbyid(defaclrole) AS owner
    FROM grants
    JOIN pg_catalog.pg_namespace nsp ON nsp.oid = defaclnamespace
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE (grantee = 0 OR rolname IS NOT NULL)
      AND nspname NOT LIKE 'pg\\_%temp\\_%'
      AND nspname <> 'pg_toast'
    -- ORDER BY 1, 2, 3, 5
    """),
    globaldefacl=dedent("""\
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
      priv AS key,
      NULL AS "schema",
      COALESCE(rolname, 'public') as rolname,
      TRUE AS "full",
      pg_catalog.pg_get_userbyid(owner) AS owner
    FROM grants
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE rolname IS NOT NULL OR grantee = 0
    """),
    nspacl=dedent("""\
    WITH grants AS (
      SELECT
        nspname,
        (aclexplode(nspacl)).grantee AS grantee,
        (aclexplode(nspacl)).privilege_type AS priv
      FROM pg_catalog.pg_namespace
    )
    SELECT
      grants.priv AS key,
      nspname,
      COALESCE(rolname, 'public') AS rolname
    FROM grants
    LEFT OUTER JOIN pg_catalog.pg_roles AS rol ON grants.grantee = rol.oid
    WHERE (grantee = 0 OR rolname IS NOT NULL)
      AND nspname NOT LIKE 'pg\\_%temp\\_%'
      AND nspname <> 'pg_toast'
    ORDER BY 1, 2
    """)
)

_datacl_tpl = dict(
    type='datacl',
    inspect=dict(shared_query='datacl', keys=['%(privilege)s']),
    grant="GRANT %(privilege)s ON DATABASE {database} TO {role};",
    revoke="REVOKE %(privilege)s ON DATABASE {database} FROM {role};",

)

_global_defacl_tpl = dict(
    type='globaldefacl',
    inspect=dict(shared_query='globaldefacl', keys=['%(privilege)s']),
    grant=(
        "ALTER DEFAULT PRIVILEGES FOR ROLE {owner}"
        " GRANT %(privilege)s ON %(TYPE)s TO {role};"),
    revoke=(
        "ALTER DEFAULT PRIVILEGES FOR ROLE {owner}"
        " REVOKE %(privilege)s ON %(TYPE)s FROM {role};"),
)

_defacl_tpl = dict(
    type="defacl",
    inspect=dict(shared_query='defacl', keys=['%(privilege)s_on_%(t)s']),
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
    inspect=dict(shared_query='nspacl', keys=['%(privilege)s']),
    grant="GRANT %(privilege)s ON SCHEMA {schema} TO {role};",
    revoke="REVOKE %(privilege)s ON SCHEMA {schema} FROM {role};",
)

# ALL TABLES is tricky because we have to manage partial grant. But the
# trickiest comes when there is no tables in a namespace. In this case, is it
# granted or revoked ? We have to tell ldap2pg that this grant is irrelevant on
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
# meaning privilege is irrelevant : it is both granted and revoked.
#
# When namespace has tables, we compare grants to availables tables to
# determine if privilege is fully granted. If the privilege is not granted at
# all, we drop the row in WHERE clause to ensure the privilege is considered as
# revoked.
#
_allrelacl_tpl = dict(
    type='nspacl',
    inspect=dedent("""\
    WITH
    namespace_rels AS (
      SELECT
        nsp.oid,
        nsp.nspname,
        array_remove(array_agg(rel.relname ORDER BY rel.relname), NULL) AS rels
      FROM pg_catalog.pg_namespace nsp
      LEFT OUTER JOIN pg_catalog.pg_class AS rel
        ON rel.relnamespace = nsp.oid AND relkind IN %(t_array)s
      WHERE nspname NOT LIKE 'pg\\_%%temp\\_%%'
        AND nspname <> 'pg_toast'
      GROUP BY 1, 2
    ),
    all_grants AS (
      SELECT
        relnamespace,
        (aclexplode(relacl)).privilege_type,
        (aclexplode(relacl)).grantee,
        array_agg(relname ORDER BY relname) AS rels
      FROM pg_catalog.pg_class
      WHERE relkind IN %(t_array)s
      GROUP BY 1, 2, 3
    ),
    all_roles AS (
      SELECT 0 AS oid, 'public' AS rolname
      UNION
      SELECT oid, rolname from pg_roles
    )
    SELECT
      nspname,
      rolname,
      CASE
        WHEN nsp.rels = ARRAY[]::name[] THEN NULL
        ELSE nsp.rels = COALESCE(grants.rels, ARRAY[]::name[])
      END AS "full"
    FROM namespace_rels AS nsp
    CROSS JOIN all_roles AS rol
    LEFT OUTER JOIN all_grants AS grants
      ON relnamespace = nsp.oid
         AND grantee = rol.oid
         AND privilege_type = '%(privilege)s'
    WHERE NOT (array_length(nsp.rels, 1) IS NOT NULL AND grants.rels IS NULL)
    -- ORDER BY 1, 2
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
        array_remove(array_agg(DISTINCT pro.proname ORDER BY pro.proname), NULL) AS procs
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
        WHEN nsp.procs = ARRAY[]::name[] THEN NULL
        ELSE nsp.procs = COALESCE(grants.procs, ARRAY[]::name[])
      END AS "full"
    FROM namespaces AS nsp
    CROSS JOIN roles
    LEFT OUTER JOIN grants
      ON pronamespace = nsp.oid AND grants.grantee = roles.oid
    WHERE NOT (array_length(nsp.procs, 1) IS NOT NULL AND grants.procs IS NULL)
      AND (priv IS NULL OR priv = '%(privilege)s')
      AND nspname NOT LIKE 'pg\\_%%temp\\_%%'
    -- ORDER BY 1, 2
    """),  # noqa
    grant="GRANT %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} TO {role}",
    revoke=(
        "REVOKE %(privilege)s ON ALL %(TYPE)s IN SCHEMA {schema} FROM {role}"),
)


_types = {
    'FUNCTIONS': ('f',),
    'TABLES': ('r', 'v', 'f'),
    'TYPES': ('T',),
    'SEQUENCES': ('S',),
}


def format_keys(fmt, fmt_kwargs):
    if '%(t)' in fmt:
        for t in fmt_kwargs['t']:
            yield fmt % dict(fmt_kwargs, t=t)
    else:
        yield fmt % fmt_kwargs


def make_privilege(tpl, name, TYPE, privilege):
    t = _types.get(TYPE)
    fmt_args = dict(
        t=t,
        # Loose SQL formatting
        t_array='(%s)' % (', '.join(['%r' % i for i in t or []])),
        TYPE=TYPE,
        privilege=privilege.upper(),
    )
    privilege = dict()
    for k, v in tpl.items():
        if isinstance(v, string_types):
            v = v % fmt_args
        else:
            if v['shared_query'] not in shared_queries:
                raise Exception("Unknown query %s." % v['shared_query'])
            v = v.copy()
            v['keys'] = list(chain(*[
                format_keys(key, fmt_args)
                for key in v['keys']
            ]))
        privilege[k] = v
    return name, privilege


def make_proc_privileges(
        privilege, TYPE='FUNCTIONS', namefmt='__%(privilege)s_on_%(type)s__'):
    fmtkw = dict(privilege=privilege.lower(), type=TYPE.lower())
    all_ = '__%(privilege)s_on_all_%(type)s__' % fmtkw
    default = '__default_%(privilege)s_on_%(type)s__' % fmtkw
    global_def = '__global_default_%(privilege)s_on_%(type)s__' % fmtkw
    name = namefmt % fmtkw
    return dict([
        make_privilege(_allprocacl_tpl, all_, TYPE, privilege),
        make_privilege(_defacl_tpl, default, TYPE, privilege),
        make_privilege(_global_defacl_tpl, global_def, TYPE, privilege),
        (name, [all_, default, global_def]),
    ])


def make_rel_privileges(
        privilege, TYPE, namefmt='__%(privilege)s_on_%(type)s__'):
    fmtkw = dict(privilege=privilege.lower(), type=TYPE.lower())
    all_ = '__%(privilege)s_on_all_%(type)s__' % fmtkw
    default = '__default_%(privilege)s_on_%(type)s__' % fmtkw
    name = namefmt % fmtkw
    return dict([
        make_privilege(_allrelacl_tpl, all_, TYPE, privilege),
        make_privilege(_defacl_tpl, default, TYPE, privilege),
        (name, [all_, default]),
    ])


def make_well_known_privileges():
    privileges = dict([
        make_privilege(_datacl_tpl, '__connect__', None, 'CONNECT'),
        make_privilege(_datacl_tpl, '__temporary__', None, 'TEMPORARY'),
        make_privilege(_nspacl_tpl, '__create_on_schemas__', None, 'CREATE'),
        make_privilege(_nspacl_tpl, '__usage_on_schemas__', None, 'USAGE'),
        make_privilege(
            _defacl_tpl, '__default_usage_on_types__', 'TYPES', 'USAGE'),
    ])

    # This is a compatibility alias.
    privileges['__usage_on_types__'] = ['__default_usage_on_types__']

    privileges.update(make_proc_privileges('EXECUTE', 'FUNCTIONS'))
    privileges['__execute__'] = ['__execute_on_functions__']

    for privilege in 'DELETE', 'INSERT', 'REFERENCES', 'TRIGGER', 'TRUNCATE':
        privileges.update(
            make_rel_privileges(privilege, 'TABLES'))
        alias = '__%s__' % (privilege.lower(),)
        privileges[alias] = ['__%s_on_tables__' % (privilege.lower(),)]

    for privilege in 'SELECT', 'UPDATE':
        privileges.update(make_rel_privileges(privilege, 'TABLES'))
        privileges.update(make_rel_privileges(privilege, 'SEQUENCES'))

    privileges.update(make_rel_privileges('USAGE', 'SEQUENCES'))

    privileges['__all_on_schemas__'] = [
        '__create_on_schemas__',
        '__usage_on_schemas__',
    ]

    privileges['__all_on_sequences__'] = [
        '__select_on_sequences__',
        '__update_on_sequences__',
        '__usage_on_sequences__',
    ]

    privileges['__all_on_tables__'] = [
        '__delete__',
        '__insert__',
        '__references__',
        '__select_on_tables__',
        '__trigger__',
        '__truncate__',
        '__update_on_tables__',
    ]

    return privileges
