WITH grants AS (
	SELECT
		nspname,
		(aclexplode(COALESCE(nspacl, acldefault('n', nspowner)))).grantor AS grantor,
		(aclexplode(COALESCE(nspacl, acldefault('n', nspowner)))).grantee AS grantee,
		(aclexplode(COALESCE(nspacl, acldefault('n', nspowner)))).privilege_type AS priv
	FROM pg_catalog.pg_namespace
)
SELECT
	'' AS owner,
	COALESCE(grantee.rolname, 'public') AS grantee,
	grants.priv AS "privilege",
	current_database() AS "database",
	grants.nspname AS "schema",
	'' AS "object",
	FALSE AS "partial"
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 1, 2, 4, 3
