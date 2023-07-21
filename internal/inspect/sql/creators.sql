SELECT nspname, array_agg(rolname ORDER BY rolname) AS creators
FROM pg_catalog.pg_namespace AS nsp
CROSS JOIN pg_catalog.pg_roles AS creator
WHERE has_schema_privilege(creator.oid, nsp.oid, 'CREATE')
  AND rolcanlogin
GROUP BY nspname;
