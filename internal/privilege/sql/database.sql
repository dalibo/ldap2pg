WITH grants AS (
	SELECT
		datname,
		(aclexplode(COALESCE(datacl, acldefault('d', datdba)))).grantor AS grantor,
		(aclexplode(COALESCE(datacl, acldefault('d', datdba)))).grantee AS grantee,
		(aclexplode(COALESCE(datacl, acldefault('d', datdba)))).privilege_type AS priv
	FROM pg_catalog.pg_database
)
SELECT
	grants.priv AS "privilege",
	grants.datname AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 1
