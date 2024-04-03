WITH me AS (
  SELECT * FROM pg_catalog.pg_roles
   WHERE rolname = CURRENT_USER
), memberships AS (
  SELECT ms.member AS member,
         p.rolname AS "name",
         g.rolname AS "grantor"
    FROM pg_auth_members AS ms
    JOIN pg_roles AS p ON p.oid = ms.roleid
    JOIN pg_roles AS g ON g.oid = ms.grantor
)
SELECT rol.rolname,
       -- Encapsulate columns variation in a sub-row.
       row(rol.*) AS opt,
       COALESCE(pg_catalog.shobj_description(rol.oid, 'pg_authid'), '') AS comment,
       -- Postgres 16 allows: json_arrayagg(memberships.* ORDER BY 2 ABSENT ON NULL)::jsonb AS parents,
			 -- may return {NULL}, array_remove can't compare json object.
       array_agg(to_json(memberships.*)) AS parents,
       rol.rolconfig AS config,
       pg_has_role(CURRENT_USER, rol.rolname, 'USAGE') AS manageable
  FROM me
       CROSS JOIN pg_catalog.pg_roles AS rol
       LEFT OUTER JOIN memberships ON memberships.member = rol.oid
 WHERE NOT (rol.rolsuper AND NOT me.rolsuper)
 GROUP BY 1, 2, 3, 5
 ORDER BY 1
