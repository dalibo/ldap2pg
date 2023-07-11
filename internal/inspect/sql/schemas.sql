SELECT nspname, rolname
FROM pg_catalog.pg_namespace
JOIN pg_catalog.pg_roles ON pg_catalog.pg_roles.oid = nspowner
-- Ensure ldap2pg can use.
WHERE has_schema_privilege(CURRENT_USER, nspname, 'USAGE')
ORDER BY 1;
