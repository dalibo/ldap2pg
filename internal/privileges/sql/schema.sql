WITH grants AS (
	SELECT
		nspname,
		grt.grantor AS grantor,
		grt.grantee AS grantee,
		grt.privilege_type AS priv
	FROM pg_catalog.pg_namespace AS nsp
  NATURAL JOIN aclexplode(COALESCE(nsp.nspacl, acldefault('n', nsp.nspowner))) AS grt
)
SELECT
	grants.priv AS "privilege",
	grants.nspname AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee,
	FALSE AS partial
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE "priv" = ANY ($1)
ORDER BY 2, 3, 1
