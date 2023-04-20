SELECT datname, rolname
FROM pg_catalog.pg_database
JOIN pg_catalog.pg_roles
  ON pg_catalog.pg_roles.oid = datdba
  -- Ensure ldap2pg can reassign to owner.
WHERE pg_has_role(CURRENT_USER, datdba, 'USAGE')
ORDER BY 1;
