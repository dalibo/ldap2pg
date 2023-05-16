SELECT attrs.attname
FROM pg_catalog.pg_namespace AS nsp
JOIN pg_catalog.pg_class AS tables
  ON tables.relnamespace = nsp.oid AND tables.relname = 'pg_roles'
JOIN pg_catalog.pg_attribute AS attrs
  ON attrs.attrelid = tables.oid AND attrs.attname LIKE 'rol%'
WHERE nsp.nspname = 'pg_catalog'
ORDER BY 1
