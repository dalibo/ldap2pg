WITH
namespace_rels AS (
	SELECT
		nsp.oid,
		nsp.nspname,
		array_remove(array_agg(rel.relname ORDER BY rel.relname), NULL) AS rels
	FROM pg_catalog.pg_namespace nsp
	LEFT OUTER JOIN pg_catalog.pg_class AS rel
		ON rel.relnamespace = nsp.oid AND relkind = 'S'
	WHERE nspname NOT LIKE 'pg\\_%temp\\_%'
		AND nspname <> 'pg_toast'
	GROUP BY 1, 2
),
grants AS (
	SELECT
		relnamespace,
		grt.privilege_type,
		grt.grantee,
		array_agg(relname ORDER BY relname) AS rels
	FROM pg_catalog.pg_class AS rel
  NATURAL JOIN aclexplode(rel.relacl) AS grt
	WHERE relkind = 'S'
	GROUP BY 1, 2, 3
)
SELECT
	COALESCE(privilege_type, '') AS "privilege",
	nspname AS "schema",
	COALESCE(rolname, 'public') AS grantee,
	nsp.rels <> COALESCE(grants.rels, ARRAY[]::name[]) AS "partial"
FROM namespace_rels AS nsp
LEFT OUTER JOIN grants AS grants
	ON relnamespace = nsp.oid
			AND privilege_type = ANY($1)
LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE NOT (array_length(nsp.rels, 1) IS NOT NULL AND grants.rels IS NULL)
ORDER BY 1, 2
