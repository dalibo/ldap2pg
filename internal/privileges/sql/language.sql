WITH grants AS (
	SELECT
		lanname,
		grt.grantor AS grantor,
		grt.grantee AS grantee,
		grt.privilege_type AS priv
	FROM pg_catalog.pg_language AS lang
  NATURAL JOIN aclexplode(COALESCE(lang.lanacl, acldefault('T', lang.lanowner))) AS grt
)
SELECT
	grants.priv AS "privilege",
	grants.lanname AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 1
