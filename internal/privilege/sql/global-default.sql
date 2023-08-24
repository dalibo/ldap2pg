WITH grants AS (
	SELECT
		defaclnamespace AS nsp,
		defaclrole AS owner,
		CASE defaclobjtype
		WHEN 'f' THEN 'FUNCTIONS'
		WHEN 'S' THEN 'SEQUENCES'
		WHEN 'r' THEN 'TABLES'
		END AS "object",
		defaclobjtype AS objtype,
		(aclexplode(defaclacl)).grantee AS grantee,
		(aclexplode(defaclacl)).privilege_type AS priv
	FROM pg_catalog.pg_default_acl
)
SELECT
  -- column order comes from statement:
	-- ALTER DEFAULT PRIVILEGES FOR $owner GRANT $privilege ON $object TO $grantee;
	COALESCE(owner.rolname, 'public') AS owner,
	grants.priv AS "privilege",
	grants."object" AS "object",
	COALESCE(grantee.rolname, 'public') AS grantee
FROM grants
LEFT OUTER JOIN pg_catalog.pg_roles AS owner ON owner.oid = grants.owner
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
LEFT OUTER JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = grants.nsp
WHERE "nspname" IS NULL			-- Handle global default privileges only.
	AND grants."object" || '--' || "priv" = ANY ($1)
ORDER BY 1, 3, 4, 2
