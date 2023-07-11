WITH grants AS (
	SELECT
		lanname,
		(aclexplode(COALESCE(lanacl, acldefault('T', lanowner)))).grantor AS grantor,
		(aclexplode(COALESCE(lanacl, acldefault('T', lanowner)))).grantee AS grantee,
		(aclexplode(COALESCE(lanacl, acldefault('T', lanowner)))).privilege_type AS priv
	FROM pg_catalog.pg_language
)
SELECT
	'' AS owner,
	COALESCE(grantee.rolname, 'public') AS grantee,
	grants.priv AS "privilege",
	'' AS "database",
	'' AS "schema",
	grants.lanname AS "object",
	FALSE AS "partial"
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 1, 2, 4, 3
