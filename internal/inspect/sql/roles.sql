WITH me AS (
  SELECT * FROM pg_catalog.pg_roles WHERE rolname = CURRENT_USER
)
SELECT
	rol.rolname,
	-- Encapsulate columns variation in a sub-row.
	row(rol.*) AS opt,
	COALESCE(pg_catalog.shobj_description(rol.oid, 'pg_authid'), '') as comment,
	array_remove(array_agg(parents.rolname), NULL) AS parents,
	rol.rolconfig AS config,
	COALESCE(ownership.admin_option, FALSE) AS manageable
FROM me
CROSS JOIN pg_catalog.pg_roles AS rol
LEFT OUTER JOIN pg_catalog.pg_auth_members AS ownership
  ON ownership.member = me.oid AND ownership.roleid = rol.oid
LEFT JOIN pg_catalog.pg_auth_members AS membership ON membership.member = rol.oid
LEFT JOIN pg_catalog.pg_roles AS parents ON parents.oid = membership.roleid
WHERE NOT (rol.rolsuper AND NOT me.rolsuper)
GROUP BY 1, 2, 3, 5, 6
ORDER BY 1;
