WITH grants AS (
	SELECT
		nspname,
		(aclexplode(COALESCE(nspacl, acldefault('n', nspowner)))).grantor AS grantor,
		(aclexplode(COALESCE(nspacl, acldefault('n', nspowner)))).grantee AS grantee,
		(aclexplode(COALESCE(nspacl, acldefault('n', nspowner)))).privilege_type AS priv
	FROM pg_catalog.pg_namespace
)
SELECT
	grants.priv AS "privilege",
	grants.nspname AS "schema",
	'' AS "object",								-- No object means schema itself.
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 4, 1
