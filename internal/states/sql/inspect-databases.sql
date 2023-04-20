SELECT datname, rolname
FROM pg_catalog.pg_database
JOIN pg_catalog.pg_roles
  ON pg_catalog.pg_roles.oid = datdba
WHERE datallowconn
ORDER BY 1;
