WITH grants AS (
	SELECT
		datname,
		grt.grantor AS grantor,
		grt.grantee AS grantee,
		grt.privilege_type AS priv
	FROM pg_catalog.pg_database AS db
  NATURAL JOIN  aclexplode(COALESCE(db.datacl, acldefault('d', db.datdba))) AS grt
)
SELECT
	grants.priv AS "privilege",
	grants.datname AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 1
