WITH grants AS (
	SELECT
		lanname,
		(aclexplode(COALESCE(lanacl, acldefault('T', lanowner)))).grantor AS grantor,
		(aclexplode(COALESCE(lanacl, acldefault('T', lanowner)))).grantee AS grantee,
		(aclexplode(COALESCE(lanacl, acldefault('T', lanowner)))).privilege_type AS priv
	FROM pg_catalog.pg_language
)
SELECT
	grants.priv AS "privilege",
	grants.lanname AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 1
