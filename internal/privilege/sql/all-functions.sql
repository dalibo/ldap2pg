WITH
grants AS (SELECT
	pronamespace, grantee, priv,
	array_agg(DISTINCT proname ORDER BY proname) AS procs
	FROM (
		SELECT
			pronamespace,
			proname,
			(aclexplode(COALESCE(proacl, acldefault('f', proowner)))).grantee,
			(aclexplode(COALESCE(proacl, acldefault('f', proowner)))).privilege_type AS priv
		FROM pg_catalog.pg_proc
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
	WHERE nspname NOT LIKE 'pg\_%temp\_%' AND nspname <> 'pg_toast'
	GROUP BY 1, 2
)
SELECT
	COALESCE(priv, '') AS "privilege",
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
