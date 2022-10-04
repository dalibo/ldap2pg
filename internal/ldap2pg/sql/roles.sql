SELECT
  rol.rolname,
  -- Encapsulate columns variation in a sub-row.
  row(rol.*) AS opt,
  COALESCE(pg_catalog.shobj_description(rol.oid, 'pg_authid'), '') as comment,
  array_remove(array_agg(members.rolname), NULL) AS members
FROM
  pg_catalog.pg_roles AS rol
LEFT JOIN pg_catalog.pg_auth_members ON member = rol.oid
LEFT JOIN pg_catalog.pg_roles AS members ON members.oid = roleid
GROUP BY 1, 2, 3
ORDER BY 1;
