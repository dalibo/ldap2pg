WITH hardwired(object, priv) AS (
    -- Postgres hardwire the following default privileges on self.
    VALUES ('ROUTINES', 'EXECUTE'),
           ('SEQUENCES', 'USAGE'),
           ('SEQUENCES', 'UPDATE'),
           ('SEQUENCES', 'SELECT'),
           ('TABLES', 'SELECT'),
           ('TABLES', 'INSERT'),
           ('TABLES', 'UPDATE'),
           ('TABLES', 'DELETE'),
           ('TABLES', 'TRUNCATE'),
           ('TABLES', 'REFERENCES'),
           ('TABLES', 'TRIGGER')
),
grants AS (
    -- Produce default privilege on self from hardwired values.
    SELECT 0::oid AS nsp,
           pg_roles.oid AS owner,
           object,
           pg_roles.oid AS grantee,
           priv
      FROM pg_catalog.pg_roles
           LEFT OUTER JOIN pg_catalog.pg_default_acl
                        ON defaclrole = pg_roles.oid
                       AND defaclnamespace = 0
     CROSS JOIN hardwired
     WHERE defaclnamespace IS NULL

     UNION ALL

     SELECT 0::oid AS nsp,
            pg_roles.oid AS owner,
            'FUNCTIONS' AS object,
            0::oid AS grantee,
            'EXECUTE' AS priv
     FROM pg_catalog.pg_roles
     LEFT OUTER JOIN pg_catalog.pg_default_acl
          ON defaclrole = pg_roles.oid
          AND defaclnamespace = 0
     WHERE defaclnamespace IS NULL

     UNION ALL

    SELECT defaclnamespace AS nsp,
           defaclrole AS owner,
           CASE defaclobjtype
           WHEN 'f' THEN 'FUNCTIONS'
           WHEN 'S' THEN 'SEQUENCES'
           WHEN 'r' THEN 'TABLES'
           END AS object,
           grt.grantee AS grantee,
           grt.privilege_type AS priv
      FROM pg_catalog.pg_default_acl AS defacl
      NATURAL JOIN aclexplode(defacl.defaclacl) AS grt
     WHERE defaclnamespace = 0
)
-- column order comes from statement:
-- ALTER DEFAULT PRIVILEGES FOR $owner GRANT $privilege ON $object TO $grantee;
SELECT COALESCE(owner.rolname, 'public') AS owner,
       grants.priv AS privilege,
       grants.object AS object,
       COALESCE(grantee.rolname, 'public') AS grantee
  FROM grants
       LEFT OUTER JOIN pg_catalog.pg_roles AS owner ON owner.oid = grants.owner
       LEFT OUTER JOIN pg_catalog.pg_roles AS grantee ON grantee.oid = grants.grantee
WHERE nsp = 0  -- Handle global default privileges only.
   AND priv || ' ON ' || grants.object = ANY ($1)
 ORDER BY 1, 3, 4, 2
