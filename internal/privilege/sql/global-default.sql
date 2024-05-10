WITH hardwired(object, priv) AS (
    -- Postgres hardwire the following default privileges on self.
    VALUES ('FUNCTIONS', 'EXECUTE'),
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

    SELECT defaclnamespace AS nsp,
           defaclrole AS owner,
           CASE defaclobjtype
           WHEN 'f' THEN 'FUNCTIONS'
           WHEN 'S' THEN 'SEQUENCES'
           WHEN 'r' THEN 'TABLES'
           END AS object,
           (aclexplode(defaclacl)).grantee AS grantee,
           (aclexplode(defaclacl)).privilege_type AS priv
      FROM pg_catalog.pg_default_acl
     WHERE defaclnamespace = 0
)
-- column order comes from statement:
-- ALTER DEFAULT PRIVILEGES FOR $owner GRANT $privilege ON $object TO $grantee;
SELECT COALESCE(grants.owner::regrole::text, 'public') AS owner,
       grants.priv AS privilege,
       grants.object AS object,
       COALESCE(grants.grantee::regrole::text, 'public') AS grantee
  FROM grants
 WHERE nsp = 0  -- Handle global default privileges only.
   AND grants.object || '--' || priv = ANY ($1)
 ORDER BY 1, 3, 4, 2
