WITH
grants AS (SELECT
	pronamespace, grantee, privilege_type,
	array_agg(DISTINCT proname ORDER BY proname) AS procs
	FROM (
		SELECT
			pronamespace,
			proname,
			grt.grantee,
			grt.privilege_type
		FROM pg_catalog.pg_proc AS pro
    NATURAL JOIN aclexplode(COALESCE(pro.proacl, acldefault('f', pro.proowner))) AS grt
    JOIN pg_catalog.pg_type AS rettype
      ON rettype.oid = pro.prorettype
    WHERE rettype.typname <> 'void'  -- skip procedures
	) AS grants
	GROUP BY 1, 2, 3
),
namespaces AS (
	SELECT
		nsp.oid, nsp.nspname,
		array_remove(array_agg(DISTINCT pro.proname ORDER BY pro.proname), NULL) AS procs
	FROM pg_catalog.pg_namespace nsp
	LEFT OUTER JOIN pg_catalog.pg_proc AS pro
		ON pro.pronamespace = nsp.oid
	LEFT OUTER JOIN pg_catalog.pg_type AS voidret
    ON voidret.oid = pro.prorettype AND voidret.typname <> 'void'
	WHERE nspname NOT LIKE 'pg\_%temp\_%' AND nspname <> 'pg_toast'
	  AND voidret.oid IS NOT NULL -- exclude procedures.
	GROUP BY 1, 2
)
SELECT
	COALESCE(privilege_type, '') AS "privilege",
	nspname AS "schema",
	COALESCE(rolname, 'public') AS grantee,
	nsp.procs <> COALESCE(grants.procs, ARRAY[]::name[]) AS "partial"
FROM namespaces AS nsp
LEFT OUTER JOIN grants
	ON pronamespace = nsp.oid
	AND privilege_type = ANY($1)
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE NOT (array_length(nsp.procs, 1) IS NOT NULL AND grants.procs IS NULL)
ORDER BY 1, 2
