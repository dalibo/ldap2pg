WITH me AS (
	SELECT
		rolname AS "current_user",
		rolsuper AS "issuper"
	FROM pg_catalog.pg_roles
	WHERE rolname = CURRENT_USER
),
postgres AS (
	SELECT
		version() AS server_version,
		svn.setting::BIGINT AS server_version_num,
		cn.setting AS cluster_name,
		current_database() AS current_database
	FROM pg_catalog.pg_settings AS svn
	JOIN pg_catalog.pg_settings AS cn
	  ON "cn"."name" = 'cluster_name'
	WHERE "svn"."name" = 'server_version_num'
)
SELECT
	postgres.*,
	me.*
FROM postgres, me;
