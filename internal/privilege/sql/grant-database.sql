WITH grants AS (
	SELECT
		datname,
		(aclexplode(COALESCE(datacl, acldefault('d', datdba)))).grantor AS grantor,
		(aclexplode(COALESCE(datacl, acldefault('d', datdba)))).grantee AS grantee,
		(aclexplode(COALESCE(datacl, acldefault('d', datdba)))).privilege_type AS priv
	FROM pg_catalog.pg_database
)
SELECT
	COALESCE(grantor.rolname, 'public') AS grantor,
	COALESCE(grantee.rolname, 'public') AS grantee,
	grants.priv AS "privilege",
	grants.datname AS "database",
	'' AS "schema",
	'' AS "object",
	FALSE AS "partial"
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantor ON grantor.oid = grants.grantor
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 1, 2, 4, 3
