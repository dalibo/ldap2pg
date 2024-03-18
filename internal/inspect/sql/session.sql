WITH me AS (
	SELECT
		rolname AS "current_user",
		rolsuper AS "issuper"
	FROM pg_catalog.pg_roles
	WHERE rolname = CURRENT_USER
),
postgres AS (
	SELECT
		substring(version() from '^[^ ]+ [^ ]+') AS server_version,
		current_setting('server_version_num')::BIGINT AS server_version_num,
		current_setting('cluster_name') AS cluster_name,
		current_database() AS current_database
)
SELECT
	postgres.*,
	me.*
FROM postgres, me;
