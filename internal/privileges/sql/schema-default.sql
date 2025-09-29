WITH grants AS (
	SELECT
		defaclnamespace AS nsp,
		defaclrole AS owner,
		CASE defaclobjtype
		WHEN 'r' THEN 'TABLES'
		WHEN 'S' THEN 'SEQUENCES'
		WHEN 'f' THEN 'ROUTINES'
		END AS "object",
		defaclobjtype AS objtype,
		(aclexplode(defaclacl)).grantee AS grantee,
		(aclexplode(defaclacl)).privilege_type AS priv
	FROM pg_catalog.pg_default_acl
)
SELECT
	COALESCE(owner.rolname, 'public') AS owner,
	"nspname" AS "schema",
	grants.priv AS "privilege",
	grants."object" AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS owner ON owner.oid = grants.owner
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
LEFT OUTER JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = grants.nsp
WHERE "nspname" IS NOT NULL			-- Handle schema default privileges only.
	AND "priv" || ' ON ' || grants."object" = ANY ($1)
ORDER BY 1, 2, 4, 3, 5
