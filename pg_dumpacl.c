/*-------------------------------------------------------------------------
 *
 * pg_dumpacl.c
 *
 * Portions Copyright (c) 2018, Dalibo
 * Portions Copyright (c) 2014, Ronan Dunklau
 * Portions Copyright (c) 1996-2017, PGDG for the pg_dumpall code
 * Portions Copyright (c) 1994, Regents of the University of California
 *
 * pg_dumpacl is a tool forked from the pg_dumpall utility by Ronan Dunklau.
 * pg_dumpacl will read the system catalogs in a database and dump out a
 * script that reproduces the database creation and ACL in terms of SQL that is
 * understood by PostgreSQL.
 *
 *-------------------------------------------------------------------------
 */

#include "postgres.h"
#include <stdlib.h>
#include <string.h>
#include "getopt_long.h"
#include "pg_config_manual.h"
#include "libpq-fe.h"
#include "pqexpbuffer.h"
#define supports_grant_options(version) ((version) >= 70400)


enum trivalue
{
	TRI_DEFAULT,
	TRI_NO,
	TRI_YES
};


char	   *pg_strdup(const char *str);
void	   *pg_malloc(size_t size);

void *
pg_malloc(size_t size)
{
	void	   *ptr;

	/* Avoid unportable behavior of malloc(0) */
	if (size == 0)
		size = 1;
	ptr = malloc(size);
	if (!ptr)
	{
		fprintf(stderr, "out of memory\n");
		exit(1);
	}
	return ptr;
}

char *
pg_strdup(const char *str)
{
	char	   *result = strdup(str);

	if (!result)
	{
		fprintf(stderr, "out of memory\n");
		exit(1);
	}
	return result;
}


static void help(void);
static PGconn *connectDatabase(const char *dbname, const char *connstr, const char *pghost, const char *pgport,
	  const char *pguser, enum trivalue prompt_password, bool fail_on_error);
static char *constructConnStr(const char **keywords, const char **values);
static void doShellQuoting(PQExpBuffer buf, const char *str);
static void doConnStrQuoting(PQExpBuffer buf, const char *str);
static bool buildACLCommands(const char *name, const char *subname,
				 const char *type, const char *acls, const char *owner,
				 const char *prefix, int remoteVersion,
				 PQExpBuffer sql);
static void makeAlterConfigCommand(PGconn *conn, const char *arrayitem,
					   const char *type, const char *name, const char *type2,
					   const char *name2);

static void AddAcl(PQExpBuffer aclbuf, const char *keyword,
	   const char *subname);

static char *copyAclUserName(PQExpBuffer output, char *input);

static bool parseAclItem(const char *item, const char *type,
			 const char *name, const char *subname, int remoteVersion,
			 PQExpBuffer grantee, PQExpBuffer grantor,
			 PQExpBuffer privs, PQExpBuffer privswgo);
bool
			parsePGArray(const char *atext, char ***itemarray, int *nitems);

const char *
			fmtId(const char *rawid);

static PQExpBuffer defaultGetLocalPQExpBuffer(void);
PQExpBuffer (*getLocalPQExpBuffer) (void) = defaultGetLocalPQExpBuffer;


static void dumpDatabaseConfig(PGconn *conn, const char *dbname);

static PGresult *executeQuery(PGconn *conn, const char *query, int numParams, const char **paramvalues);

void appendStringLiteral(PQExpBuffer buf, const char *str,
					int encoding, bool std_strings);
void appendStringLiteralConn(PQExpBuffer buf, const char *str,
						PGconn *conn);


static void dumpCreateDB(PGconn *conn, const char *dbname, bool dump_all_db);

static char *progname;
static PQExpBuffer pgdumpopts;
static char *connstr = "";
static char *connstr_dbname = NULL;
static FILE *OPF;
static char *filename = NULL;

static int	server_version;



int
main(int argc, char *argv[])
{
	static struct option long_options[] = {
		{"all", no_argument, NULL, 'a'},
		{"connstr", required_argument, NULL, 'c'},
		{"database", required_argument, NULL, 'd'},
		{"file", required_argument, NULL, 'f'},
		{"host", required_argument, NULL, 'h'},
		{"port", required_argument, NULL, 'p'},
		{"username", required_argument, NULL, 'U'},
		{"no-password", no_argument, NULL, 'w'},
		{"password", no_argument, NULL, 'W'},
		{NULL, 0, NULL, 0}
	};
	char	   *pghost = NULL;
	char	   *pgport = NULL;
	char	   *pguser = NULL;
	char	   *pgdb = NULL;
	bool     dump_all_db = false;
	enum trivalue prompt_password = TRI_DEFAULT;
	PGconn	   *conn;
	int			c;
	int			optindex;

	progname = argv[0];

	if (argc > 1)
	{
		if (strcmp(argv[1], "--help") == 0 || strcmp(argv[1], "-?") == 0)
		{
			help();
			exit(0);
		}
	}

	pgdumpopts = createPQExpBuffer();

	while ((c = getopt_long(argc, argv, "ac:d:f:h:l:p:U:wW", long_options, &optindex)) != -1)
	{
		switch (c)
		{
			case 'a':
				dump_all_db = true;
				break;

			case 'c':
				connstr = pg_strdup(optarg);
				break;

			case 'd':
				pgdb = pg_strdup(optarg);
				break;

			case 'f':
				filename = pg_strdup(optarg);
				appendPQExpBuffer(pgdumpopts, " -f ");
				doShellQuoting(pgdumpopts, filename);
				break;

			case 'h':
				pghost = pg_strdup(optarg);
				break;

			case 'p':
				pgport = pg_strdup(optarg);
				break;

			case 'U':
				pguser = pg_strdup(optarg);
				break;

			case 'w':
				prompt_password = TRI_NO;
				appendPQExpBuffer(pgdumpopts, " -w");
				break;

			case 'W':
				prompt_password = TRI_YES;
				appendPQExpBuffer(pgdumpopts, " -W");
				break;

			case 0:
				break;

			default:
				fprintf(stderr, _("Try \"%s --help\" for more information.\n"), progname);
				exit(1);
		}
	}

	if (strlen(connstr) != 0)
	{
		conn = connectDatabase(pgdb, connstr, pghost, pgport, pguser,
							   prompt_password, false);

		/*
		 * If pgdb is not given using the -d switch, try to get it from the
		 * connection string (connstr_dbname could be set in connectDatabase)
		 */
		if (connstr_dbname && !pgdb) {
			pgdb = strdup(connstr_dbname);
		}

		/*
		 * if we still do not have a pgdb, get it from the environment variable
		 * PGDATABASE
		 */
		if (!pgdb && getenv("PGDATABASE")) {
			pgdb = getenv("PGDATABASE");
		}

		if (!conn)
		{
			fprintf(stderr, _("%s: could not connect to database \"%s\"\n"),
					progname, pgdb);
			exit(1);
		}
	}
	else
	{
		conn = connectDatabase(pgdb, connstr, pghost, pgport, pguser,
							   prompt_password, false);
		/*
		 * if we do not have a pgdb, get it from the environment variable
		 * PGDATABASE
		 */
		if (!conn && !pgdb && getenv("PGDATABASE")) {
			pgdb = getenv("PGDATABASE");
			conn = connectDatabase(pgdb, connstr, pghost, pgport, pguser,
			                       prompt_password, false);
		}
		if (!conn)
			conn = connectDatabase("postgres", connstr, pghost, pgport, pguser,
								   prompt_password, true);
		if (!conn)
			conn = connectDatabase("template1", connstr, pghost, pgport, pguser,
								   prompt_password, true);

		if (!conn)
		{
			fprintf(stderr, _("%s: could not connect to either database \'%s\', "
			                  "\"postgres\" or \"template1\".\n"
			                  "Please specify an alternative database.\n"),
			        progname, pgdb);
			fprintf(stderr, _("Try \"%s --help\" for more information.\n"),
					progname);
			exit(1);
		}
	}

	if (filename)
	{
		OPF = fopen(filename, PG_BINARY_W);
		if (!OPF)
		{
			fprintf(stderr, _("%s: could not open the output file \"%s\": %s\n"),
					progname, filename, strerror(errno));
			exit(1);
		}
	}
	else
		OPF = stdout;

	if (!conn)
	{
		fprintf(stderr, _("%s: could not connect to database \"%s\"\n"),
				progname, pgdb);
		exit(1);
	}

	dumpCreateDB(conn, pgdb, dump_all_db);
	if (filename)
		fclose(OPF);
	exit(0);
};

static void
doShellQuoting(PQExpBuffer buf, const char *str)
{
	const char *p;

#ifndef WIN32
	appendPQExpBufferChar(buf, '\'');
	for (p = str; *p; p++)
	{
		if (*p == '\'')
			appendPQExpBufferStr(buf, "'\"'\"'");
		else
			appendPQExpBufferChar(buf, *p);
	}
	appendPQExpBufferChar(buf, '\'');
#else							/* WIN32 */

	appendPQExpBufferChar(buf, '"');
	for (p = str; *p; p++)
	{
		if (*p == '"')
			appendPQExpBufferStr(buf, "\\\"");
		else
			appendPQExpBufferChar(buf, *p);
	}
	appendPQExpBufferChar(buf, '"');
#endif   /* WIN32 */
}

static void
help(void)
{
	printf(_("%s extracts a PostgreSQL database ACLs into an SQL script file.\n\n"), progname);
	printf(_("Usage:\n"));
	printf(_("  %s [OPTION]...\n"), progname);

	printf(_("\nGeneral options:\n"));
	printf(_("  -a, --all                dump ACL for all databases\n"));
	printf(_("  -f, --file=FILENAME      output file name\n"));
	printf(_("  -?, --help               show this help, then exit\n"));

	printf(_("\nConnection options:\n"));
	printf(_("  -c, --connstr=CONNSTR    connect using connection string\n"));
	printf(_("  -h, --host=HOSTNAME      database server host or socket directory\n"));
	printf(_("  -d, --database=DBNAME    alternative default database\n"));
	printf(_("  -p, --port=PORT          database server port number\n"));
	printf(_("  -U, --username=NAME      connect as specified database user\n"));
	printf(_("  -w, --no-password        never prompt for password\n"));
	printf(_("  -W, --password           force password prompt (should happen automatically)\n"));

	printf(_("\nIf -f/--file is not used, then the SQL script will be written to the standard\n"
			 "output.\n\n"));
}

static PGconn *
connectDatabase(const char *dbname, const char *connection_string,
				const char *pghost, const char *pgport, const char *pguser,
				enum trivalue prompt_password, bool fail_on_error)
{
	PGconn	   *conn;
	bool		new_pass;
	const char *remoteversion_str;
	int			my_version;
	// API change in version 10 : no need to free password anymore
	// but we must allocate it
#if PG_VERSION_NUM > 100000
	char *password = pg_malloc(101);
#else
	static char *password = NULL;
#endif
	const char **keywords = NULL;
	const char **values = NULL;
	PQconninfoOption *conn_opts = NULL;
	static bool have_password = false;

	if (prompt_password == TRI_YES) {
#if PG_VERSION_NUM > 100000
		simple_prompt("Password: ", password, 100, false);
#else
		if (!password)
			password = simple_prompt("Password: ", 100, false);
#endif
		have_password = true;
	}

	/*
	 * Start the connection.  Loop until we have a password if requested by
	 * backend.
	 */
	do
	{
		int			argcount = 6;
		PQconninfoOption *conn_opt;
		char	   *err_msg = NULL;
		int			i = 0;

		if (keywords)
			free(keywords);
		if (values)
			free(values);
		if (conn_opts)
			PQconninfoFree(conn_opts);

		/*
		 * Merge the connection info inputs given in form of connection string
		 * and other options. Explicitly discard any dbname value in the
		 * connection string; otherwise, PQconnectdbParams() would interpret
		 * that value as being itself a connection string.*/
		if (connection_string)
		{
			conn_opts = PQconninfoParse(connection_string, &err_msg);

			if (conn_opts == NULL)
			{
				fprintf(stderr, "%s: %s", progname, err_msg);
				exit(1);
			}

			for (conn_opt = conn_opts; conn_opt->keyword != NULL; conn_opt++)
			{
				if (conn_opt->val != NULL && conn_opt->val[0] != '\0')
					argcount++;
			}

			keywords = pg_malloc((argcount + 1) * sizeof(*keywords));
			values = pg_malloc((argcount + 1) * sizeof(*values));

			for (conn_opt = conn_opts; conn_opt->keyword != NULL; conn_opt++)
			{
				if (conn_opt->val != NULL && conn_opt->val[0] != '\0')
				{
					/*
					 * if the dbname was given in the connstr and not in pgdb using the -d
					 * switch, we must retrieve it and save it in the pgdb variable for
					 * the dumpCreateDB operation
					 */
					if (strcmp(conn_opt->keyword, "dbname") == 0 && !dbname)
						connstr_dbname = pg_strdup(conn_opt->val);

					keywords[i] = conn_opt->keyword;
					values[i] = conn_opt->val;
					i++;
				}
			}
		}
		else
		{
			keywords = pg_malloc((argcount + 1) * sizeof(*keywords));
			values = pg_malloc((argcount + 1) * sizeof(*values));
		}

		if (pghost)
		{
			keywords[i] = "host";
			values[i] = pghost;
			i++;
		}
		if (pgport)
		{
			keywords[i] = "port";
			values[i] = pgport;
			i++;
		}
		if (pguser)
		{
			keywords[i] = "user";
			values[i] = pguser;
			i++;
		}
		if (have_password)
		{
			keywords[i] = "password";
			values[i] = password;
			i++;
		}
		if (dbname)
		{
			keywords[i] = "dbname";
			values[i] = dbname;
			i++;
		}
		keywords[i] = "fallback_application_name";
		values[i] = progname;
		i++;

		new_pass = false;
		conn = PQconnectdbParams(keywords, values, true);

		if (!conn)
		{
			fprintf(stderr, _("%s: could not connect to database \"%s\"\n"),
					progname, dbname);
			exit(1);
		}

		if (PQstatus(conn) == CONNECTION_BAD &&
			PQconnectionNeedsPassword(conn) &&
			!have_password &&
			prompt_password != TRI_NO)
		{
			PQfinish(conn);

#if PG_VERSION_NUM > 100000
			simple_prompt("Password: ", password, 100, false);
#else
			password = simple_prompt("Password: ", 100, false);
#endif
			have_password = true;
			new_pass = true;
		}
	} while (new_pass);

	/* check to see that the backend connection was successfully made */
	if (PQstatus(conn) == CONNECTION_BAD)
	{
		if (fail_on_error)
		{
			fprintf(stderr,
					_("%s: could not connect to database \"%s\": %s\n"),
					progname, dbname, PQerrorMessage(conn));
			exit(1);
		}
		else
		{
			PQfinish(conn);

			free(keywords);
			free(values);
			PQconninfoFree(conn_opts);

			return NULL;
		}
	}

	/*
	 * Ok, connected successfully. Remember the options used, in the form of a
	 * connection string.
	 */
	connstr = constructConnStr(keywords, values);

	free(keywords);
	free(values);
	PQconninfoFree(conn_opts);

	/* Check version */
	remoteversion_str = PQparameterStatus(conn, "server_version");
	if (!remoteversion_str)
	{
		fprintf(stderr, _("%s: could not get server version\n"), progname);
		exit(1);
	}
	server_version = PQserverVersion(conn);
	if (server_version == 0)
	{
		fprintf(stderr, _("%s: could not parse server version \"%s\"\n"),
				progname, remoteversion_str);
		exit(1);
	}

	my_version = PG_VERSION_NUM;

	/*
	 * We allow the server to be back to 8.0, and up to any minor release of
	 * our own major version.  (See also version check in pg_dump.c.)
	 */
	if (my_version != server_version
		&& (server_version < 80000 ||
			(server_version / 100) > (my_version / 100)))
	{
		fprintf(stderr, _("server version: %s; %s version: %s\n"),
				remoteversion_str, progname, PG_VERSION);
		fprintf(stderr, _("aborting because of server version mismatch\n"));
		exit(1);
	}

	return conn;
}

static char *
constructConnStr(const char **keywords, const char **values)
{
	PQExpBuffer buf = createPQExpBuffer();
	char	   *connstr;
	int			i;
	bool		firstkeyword = true;

	/* Construct a new connection string in key='value' format. */
	for (i = 0; keywords[i] != NULL; i++)
	{
		if (strcmp(keywords[i], "dbname") == 0 ||
			strcmp(keywords[i], "password") == 0 ||
			strcmp(keywords[i], "fallback_application_name") == 0)
			continue;

		if (!firstkeyword)
			appendPQExpBufferChar(buf, ' ');
		firstkeyword = false;
		appendPQExpBuffer(buf, "%s=", keywords[i]);
		doConnStrQuoting(buf, values[i]);
	}

	connstr = pg_strdup(buf->data);
	destroyPQExpBuffer(buf);
	return connstr;
}

static void
doConnStrQuoting(PQExpBuffer buf, const char *str)
{
	const char *s;
	bool		needquotes;

	/*
	 * If the string consists entirely of plain ASCII characters, no need to
	 * quote it. This is quite conservative, but better safe than sorry.
	 */
	needquotes = false;
	for (s = str; *s; s++)
	{
		if (!((*s >= 'a' && *s <= 'z') || (*s >= 'A' && *s <= 'Z') ||
			  (*s >= '0' && *s <= '9') || *s == '_' || *s == '.'))
		{
			needquotes = true;
			break;
		}
	}

	if (needquotes)
	{
		appendPQExpBufferChar(buf, '\'');
		while (*str)
		{
			/* ' and \ must be escaped by to \' and \\ */
			if (*str == '\'' || *str == '\\')
				appendPQExpBufferChar(buf, '\\');

			appendPQExpBufferChar(buf, *str);
			str++;
		}
		appendPQExpBufferChar(buf, '\'');
	}
	else
		appendPQExpBufferStr(buf, str);
}

static void
dumpCreateDB(PGconn *conn, const char *dbname, bool dump_all_db)
{
	PQExpBuffer buf = createPQExpBuffer();
	char	   *default_encoding = NULL;
	char	   *default_collate = NULL;
	char	   *default_ctype = NULL;
	PGresult   *res;
	int			i;

	fprintf(OPF, "--\n-- Database creation\n--\n\n");

	/*
	 * First, get the installation's default encoding and locale information.
	 * We will dump encoding and locale specifications in the CREATE DATABASE
	 * commands for just those databases with values different from defaults.
	 *
	 * We consider template0's encoding and locale (or, pre-7.1, template1's)
	 * to define the installation default.	Pre-8.4 installations do not have
	 * per-database locale settings; for them, every database must necessarily
	 * be using the installation default, so there's no need to do anything
	 * (which is good, since in very old versions there is no good way to find
	 * out what the installation locale is anyway...)
	 */
	res = executeQuery(conn,
					   "SELECT pg_encoding_to_char(encoding), "
					   "datcollate, datctype "
					   "FROM pg_database "
					   "WHERE datname = 'template0'", 0, NULL);

	/* If for some reason the template DB isn't there, treat as unknown */
	if (PQntuples(res) > 0)
	{
		if (!PQgetisnull(res, 0, 0))
			default_encoding = pg_strdup(PQgetvalue(res, 0, 0));
		if (!PQgetisnull(res, 0, 1))
			default_collate = pg_strdup(PQgetvalue(res, 0, 1));
		if (!PQgetisnull(res, 0, 2))
			default_ctype = pg_strdup(PQgetvalue(res, 0, 2));
	}

	PQclear(res);

	appendPQExpBuffer(buf,
	                  "SELECT datname, "
	                  "coalesce(rolname, (select rolname from pg_authid where oid=(select datdba from pg_database where datname='template0'))), "
	                  "pg_encoding_to_char(d.encoding), "
	                  "datcollate, datctype, datfrozenxid, datminmxid, "
	                  "datistemplate, datacl, datconnlimit, "
	                  "(SELECT spcname FROM pg_tablespace t WHERE t.oid = d.dattablespace) AS dattablespace "
	                  "FROM pg_database d LEFT JOIN pg_authid u ON (datdba = u.oid) "
	                  "WHERE datallowconn");
	if (! dump_all_db)
	{
		appendPQExpBuffer(buf, " AND datname = \'%s\'", dbname);
	}
	appendPQExpBuffer(buf, " ORDER BY 1");

	res = executeQuery(conn, buf->data, 0, NULL);

	for (i = 0; i < PQntuples(res); i++)
	{
		char	   *dbname = PQgetvalue(res, i, 0);
		char	   *dbowner = PQgetvalue(res, i, 1);
		char	   *dbencoding = PQgetvalue(res, i, 2);
		char	   *dbcollate = PQgetvalue(res, i, 3);
		char	   *dbctype = PQgetvalue(res, i, 4);
		char	   *dbistemplate = PQgetvalue(res, i, 7);
		char	   *dbacl = PQgetvalue(res, i, 8);
		char	   *dbconnlimit = PQgetvalue(res, i, 9);
		char	   *dbtablespace = PQgetvalue(res, i, 10);
		char	   *fdbname;

		fdbname = pg_strdup(fmtId(dbname));

		resetPQExpBuffer(buf);

		/*
		 * Skip the CREATE DATABASE commands for "template1" and "postgres",
		 * since they are presumably already there in the destination cluster.
		 * We do want to emit their ACLs and config options if any, however.
		 */
		if (strcmp(dbname, "template1") != 0 &&
			strcmp(dbname, "postgres") != 0)
		{
			appendPQExpBuffer(buf, "CREATE DATABASE %s", fdbname);

			appendPQExpBuffer(buf, " WITH TEMPLATE = template0");

			if (strlen(dbowner) != 0)
				appendPQExpBuffer(buf, " OWNER = %s", fmtId(dbowner));

			if (default_encoding && strcmp(dbencoding, default_encoding) != 0)
			{
				appendPQExpBuffer(buf, " ENCODING = ");
				appendStringLiteralConn(buf, dbencoding, conn);
			}

			if (default_collate && strcmp(dbcollate, default_collate) != 0)
			{
				appendPQExpBuffer(buf, " LC_COLLATE = ");
				appendStringLiteralConn(buf, dbcollate, conn);
			}

			if (default_ctype && strcmp(dbctype, default_ctype) != 0)
			{
				appendPQExpBuffer(buf, " LC_CTYPE = ");
				appendStringLiteralConn(buf, dbctype, conn);
			}

			/*
			 * Output tablespace if it isn't the default.  For default, it
			 * uses the default from the template database.  If tablespace is
			 * specified and tablespace creation failed earlier, (e.g. no such
			 * directory), the database creation will fail too.  One solution
			 * would be to use 'SET default_tablespace' like we do in pg_dump
			 * for setting non-default database locations.
			 */
			if (strcmp(dbtablespace, "pg_default") != 0)
				appendPQExpBuffer(buf, " TABLESPACE = %s",
								  fmtId(dbtablespace));

			if (strcmp(dbconnlimit, "-1") != 0)
				appendPQExpBuffer(buf, " CONNECTION LIMIT = %s",
								  dbconnlimit);

			appendPQExpBuffer(buf, ";\n");

			if (strcmp(dbistemplate, "t") == 0)
			{
				appendPQExpBuffer(buf, "UPDATE pg_catalog.pg_database SET datistemplate = 't' WHERE datname = ");
				appendStringLiteralConn(buf, dbname, conn);
				appendPQExpBuffer(buf, ";\n");
			}

		}

		if (!buildACLCommands(fdbname, NULL, "DATABASE", dbacl, dbowner,
							  "", server_version, buf))
		{
			fprintf(stderr, _("%s: could not parse ACL list (%s) for database \"%s\"\n"),
					progname, dbacl, fdbname);
			PQfinish(conn);
			exit(1);
		}

		fprintf(OPF, "%s", buf->data);

		if (server_version >= 70300)
			dumpDatabaseConfig(conn, dbname);

		free(fdbname);
	}

	PQclear(res);
	destroyPQExpBuffer(buf);

	fprintf(OPF, "\n\n");
}

static PGresult *
executeQuery(PGconn *conn, const char *query, int numparams, const char **paramvalues)
{
	PGresult   *res;

	res = PQexecParams(conn, query, numparams, NULL, paramvalues, NULL, NULL, 0);
	if (!res ||
		PQresultStatus(res) != PGRES_TUPLES_OK)
	{
		fprintf(stderr, _("%s: query failed: %s"),
				progname, PQerrorMessage(conn));
		fprintf(stderr, _("%s: query was: %s\n"),
				progname, query);
		PQfinish(conn);
		exit(1);
	}

	return res;
}

void
appendStringLiteral(PQExpBuffer buf, const char *str,
					int encoding, bool std_strings)
{
	size_t		length = strlen(str);
	const char *source = str;
	char	   *target;

	if (!enlargePQExpBuffer(buf, 2 * length + 2))
		return;

	target = buf->data + buf->len;
	*target++ = '\'';

	while (*source != '\0')
	{
		char		c = *source;
		int			len;
		int			i;

		/* Fast path for plain ASCII */
		if (!IS_HIGHBIT_SET(c))
		{
			/* Apply quoting if needed */
			if (SQL_STR_DOUBLE(c, !std_strings))
				*target++ = c;
			/* Copy the character */
			*target++ = c;
			source++;
			continue;
		}

		/* Slow path for possible multibyte characters */
		len = PQmblen(source, encoding);

		/* Copy the character */
		for (i = 0; i < len; i++)
		{
			if (*source == '\0')
				break;
			*target++ = *source++;
		}

		/*
		 * If we hit premature end of string (ie, incomplete multibyte
		 * character), try to pad out to the correct length with spaces. We
		 * may not be able to pad completely, but we will always be able to
		 * insert at least one pad space (since we'd not have quoted a
		 * multibyte character).  This should be enough to make a string that
		 * the server will error out on.
		 */
		if (i < len)
		{
			char	   *stop = buf->data + buf->maxlen - 2;

			for (; i < len; i++)
			{
				if (target >= stop)
					break;
				*target++ = ' ';
			}
			break;
		}
	}

	/* Write the terminating quote and NUL character. */
	*target++ = '\'';
	*target = '\0';

	buf->len = target - buf->data;
}

void
appendStringLiteralConn(PQExpBuffer buf, const char *str, PGconn *conn)
{
	size_t		length = strlen(str);

	/*
	 * XXX This is a kluge to silence escape_string_warning in our utility
	 * programs.  It should go away someday.
	 */
	if (strchr(str, '\\') != NULL && PQserverVersion(conn) >= 80100)
	{
		/* ensure we are not adjacent to an identifier */
		if (buf->len > 0 && buf->data[buf->len - 1] != ' ')
			appendPQExpBufferChar(buf, ' ');
		appendPQExpBufferChar(buf, ESCAPE_STRING_SYNTAX);
		appendStringLiteral(buf, str, PQclientEncoding(conn), false);
		return;
	}
	/* XXX end kluge */

	if (!enlargePQExpBuffer(buf, 2 * length + 2))
		return;
	appendPQExpBufferChar(buf, '\'');
	buf->len += PQescapeStringConn(conn, buf->data + buf->len,
								   str, length, NULL);
	appendPQExpBufferChar(buf, '\'');
}

bool
buildACLCommands(const char *name, const char *subname,
				 const char *type, const char *acls, const char *owner,
				 const char *prefix, int remoteVersion,
				 PQExpBuffer sql)
{
	char	  **aclitems;
	int			naclitems;
	int			i;
	PQExpBuffer grantee,
				grantor,
				privs,
				privswgo;
	PQExpBuffer firstsql,
				secondsql;
	bool		found_owner_privs = false;

	if (strlen(acls) == 0)
		return true;			/* object has default permissions */

	/* treat empty-string owner same as NULL */
	if (owner && *owner == '\0')
		owner = NULL;

	if (!parsePGArray(acls, &aclitems, &naclitems))
	{
		if (aclitems)
			free(aclitems);
		return false;
	}

	grantee = createPQExpBuffer();
	grantor = createPQExpBuffer();
	privs = createPQExpBuffer();
	privswgo = createPQExpBuffer();

	/*
	 * At the end, these two will be pasted together to form the result. But
	 * the owner privileges need to go before the other ones to keep the
	 * dependencies valid.	In recent versions this is normally the case, but
	 * in old versions they come after the PUBLIC privileges and that results
	 * in problems if we need to run REVOKE on the owner privileges.
	 */
	firstsql = createPQExpBuffer();
	secondsql = createPQExpBuffer();

	/*
	 * Always start with REVOKE ALL FROM PUBLIC, so that we don't have to
	 * wire-in knowledge about the default public privileges for different
	 * kinds of objects.
	 */
	appendPQExpBuffer(firstsql, "%sREVOKE ALL", prefix);
	if (subname)
		appendPQExpBuffer(firstsql, "(%s)", subname);
	appendPQExpBuffer(firstsql, " ON %s %s FROM PUBLIC;\n", type, name);

	/*
	 * We still need some hacking though to cover the case where new default
	 * public privileges are added in new versions: the REVOKE ALL will revoke
	 * them, leading to behavior different from what the old version had,
	 * which is generally not what's wanted.  So add back default privs if the
	 * source database is too old to have had that particular priv.
	 */
	if (remoteVersion < 80200 && strcmp(type, "DATABASE") == 0)
	{
		/* database CONNECT priv didn't exist before 8.2 */
		appendPQExpBuffer(firstsql, "%sGRANT CONNECT ON %s %s TO PUBLIC;\n",
						  prefix, type, name);
	}

	/* Scan individual ACL items */
	for (i = 0; i < naclitems; i++)
	{
		if (!parseAclItem(aclitems[i], type, name, subname, remoteVersion,
						  grantee, grantor, privs, privswgo))
		{
			free(aclitems);
			return false;
		}

		if (grantor->len == 0 && owner)
			printfPQExpBuffer(grantor, "%s", owner);

		if (privs->len > 0 || privswgo->len > 0)
		{
			if (owner
				&& strcmp(grantee->data, owner) == 0
				&& strcmp(grantor->data, owner) == 0)
			{
				found_owner_privs = true;

				/*
				 * For the owner, the default privilege level is ALL WITH
				 * GRANT OPTION (only ALL prior to 7.4).
				 */
				if (supports_grant_options(remoteVersion)
					? strcmp(privswgo->data, "ALL") != 0
					: strcmp(privs->data, "ALL") != 0)
				{
					appendPQExpBuffer(firstsql, "%sREVOKE ALL", prefix);
					if (subname)
						appendPQExpBuffer(firstsql, "(%s)", subname);
					appendPQExpBuffer(firstsql, " ON %s %s FROM %s;\n",
									  type, name, fmtId(grantee->data));
					if (privs->len > 0)
						appendPQExpBuffer(firstsql,
										  "%sGRANT %s ON %s %s TO %s;\n",
										  prefix, privs->data, type, name,
										  fmtId(grantee->data));
					if (privswgo->len > 0)
						appendPQExpBuffer(firstsql,
							"%sGRANT %s ON %s %s TO %s WITH GRANT OPTION;\n",
										  prefix, privswgo->data, type, name,
										  fmtId(grantee->data));
				}
			}
			else
			{
				/*
				 * Otherwise can assume we are starting from no privs.
				 */
				if (grantor->len > 0
					&& (!owner || strcmp(owner, grantor->data) != 0))
					appendPQExpBuffer(secondsql, "SET SESSION AUTHORIZATION %s;\n",
									  fmtId(grantor->data));

				if (privs->len > 0)
				{
					appendPQExpBuffer(secondsql, "%sGRANT %s ON %s %s TO ",
									  prefix, privs->data, type, name);
					if (grantee->len == 0)
						appendPQExpBuffer(secondsql, "PUBLIC;\n");
					else if (strncmp(grantee->data, "group ",
									 strlen("group ")) == 0)
						appendPQExpBuffer(secondsql, "GROUP %s;\n",
									fmtId(grantee->data + strlen("group ")));
					else
						appendPQExpBuffer(secondsql, "%s;\n", fmtId(grantee->data));
				}
				if (privswgo->len > 0)
				{
					appendPQExpBuffer(secondsql, "%sGRANT %s ON %s %s TO ",
									  prefix, privswgo->data, type, name);
					if (grantee->len == 0)
						appendPQExpBuffer(secondsql, "PUBLIC");
					else if (strncmp(grantee->data, "group ",
									 strlen("group ")) == 0)
						appendPQExpBuffer(secondsql, "GROUP %s",
									fmtId(grantee->data + strlen("group ")));
					else
						appendPQExpBuffer(secondsql, "%s", fmtId(grantee->data));
					appendPQExpBuffer(secondsql, " WITH GRANT OPTION;\n");
				}

				if (grantor->len > 0
					&& (!owner || strcmp(owner, grantor->data) != 0))
					appendPQExpBuffer(secondsql, "RESET SESSION AUTHORIZATION;\n");
			}
		}
	}

	/*
	 * If we didn't find any owner privs, the owner must have revoked 'em all
	 */
	if (!found_owner_privs && owner)
	{
		appendPQExpBuffer(firstsql, "%sREVOKE ALL", prefix);
		if (subname)
			appendPQExpBuffer(firstsql, "(%s)", subname);
		appendPQExpBuffer(firstsql, " ON %s %s FROM %s;\n",
						  type, name, fmtId(owner));
	}

	destroyPQExpBuffer(grantee);
	destroyPQExpBuffer(grantor);
	destroyPQExpBuffer(privs);
	destroyPQExpBuffer(privswgo);

	appendPQExpBuffer(sql, "%s%s", firstsql->data, secondsql->data);
	destroyPQExpBuffer(firstsql);
	destroyPQExpBuffer(secondsql);

	free(aclitems);

	return true;
}

static bool
parseAclItem(const char *item, const char *type,
			 const char *name, const char *subname, int remoteVersion,
			 PQExpBuffer grantee, PQExpBuffer grantor,
			 PQExpBuffer privs, PQExpBuffer privswgo)
{
	char	   *buf;
	bool		all_with_go = true;
	bool		all_without_go = true;
	char	   *eqpos;
	char	   *slpos;
	char	   *pos;

	buf = strdup(item);
	if (!buf)
		return false;

	/* user or group name is string up to = */
	eqpos = copyAclUserName(grantee, buf);
	if (*eqpos != '=')
	{
		free(buf);
		return false;
	}

	/* grantor may be listed after / */
	slpos = strchr(eqpos + 1, '/');
	if (slpos)
	{
		*slpos++ = '\0';
		slpos = copyAclUserName(grantor, slpos);
		if (*slpos != '\0')
		{
			free(buf);
			return false;
		}
	}
	else
		resetPQExpBuffer(grantor);

	/* privilege codes */
#define CONVERT_PRIV(code, keywd) \
do { \
	if ((pos = strchr(eqpos + 1, code))) \
	{ \
		if (*(pos + 1) == '*') \
		{ \
			AddAcl(privswgo, keywd, subname); \
			all_without_go = false; \
		} \
		else \
		{ \
			AddAcl(privs, keywd, subname); \
			all_with_go = false; \
		} \
	} \
	else \
		all_with_go = all_without_go = false; \
} while (0)

	resetPQExpBuffer(privs);
	resetPQExpBuffer(privswgo);

	if (strcmp(type, "TABLE") == 0 || strcmp(type, "SEQUENCE") == 0 ||
		strcmp(type, "TABLES") == 0 || strcmp(type, "SEQUENCES") == 0)
	{
		CONVERT_PRIV('r', "SELECT");

		if (strcmp(type, "SEQUENCE") == 0 ||
			strcmp(type, "SEQUENCES") == 0)
			/* sequence only */
			CONVERT_PRIV('U', "USAGE");
		else
		{
			/* table only */
			CONVERT_PRIV('a', "INSERT");
			if (remoteVersion >= 70200)
				CONVERT_PRIV('x', "REFERENCES");
			/* rest are not applicable to columns */
			if (subname == NULL)
			{
				if (remoteVersion >= 70200)
				{
					CONVERT_PRIV('d', "DELETE");
					CONVERT_PRIV('t', "TRIGGER");
				}
				if (remoteVersion >= 80400)
					CONVERT_PRIV('D', "TRUNCATE");
			}
		}

		/* UPDATE */
		if (remoteVersion >= 70200 ||
			strcmp(type, "SEQUENCE") == 0 ||
			strcmp(type, "SEQUENCES") == 0)
			CONVERT_PRIV('w', "UPDATE");
		else
			/* 7.0 and 7.1 have a simpler worldview */
			CONVERT_PRIV('w', "UPDATE,DELETE");
	}
	else if (strcmp(type, "FUNCTION") == 0 ||
			 strcmp(type, "FUNCTIONS") == 0)
		CONVERT_PRIV('X', "EXECUTE");
	else if (strcmp(type, "LANGUAGE") == 0)
		CONVERT_PRIV('U', "USAGE");
	else if (strcmp(type, "SCHEMA") == 0)
	{
		CONVERT_PRIV('C', "CREATE");
		CONVERT_PRIV('U', "USAGE");
	}
	else if (strcmp(type, "DATABASE") == 0)
	{
		CONVERT_PRIV('C', "CREATE");
		CONVERT_PRIV('c', "CONNECT");
		CONVERT_PRIV('T', "TEMPORARY");
	}
	else if (strcmp(type, "TABLESPACE") == 0)
		CONVERT_PRIV('C', "CREATE");
	else if (strcmp(type, "TYPE") == 0 ||
			 strcmp(type, "TYPES") == 0)
		CONVERT_PRIV('U', "USAGE");
	else if (strcmp(type, "FOREIGN DATA WRAPPER") == 0)
		CONVERT_PRIV('U', "USAGE");
	else if (strcmp(type, "FOREIGN SERVER") == 0)
		CONVERT_PRIV('U', "USAGE");
	else if (strcmp(type, "FOREIGN TABLE") == 0)
		CONVERT_PRIV('r', "SELECT");
	else if (strcmp(type, "LARGE OBJECT") == 0)
	{
		CONVERT_PRIV('r', "SELECT");
		CONVERT_PRIV('w', "UPDATE");
	}
	else
		abort();

#undef CONVERT_PRIV

	if (all_with_go)
	{
		resetPQExpBuffer(privs);
		printfPQExpBuffer(privswgo, "ALL");
		if (subname)
			appendPQExpBuffer(privswgo, "(%s)", subname);
	}
	else if (all_without_go)
	{
		resetPQExpBuffer(privswgo);
		printfPQExpBuffer(privs, "ALL");
		if (subname)
			appendPQExpBuffer(privs, "(%s)", subname);
	}

	free(buf);

	return true;
}

static void
AddAcl(PQExpBuffer aclbuf, const char *keyword, const char *subname)
{
	if (aclbuf->len > 0)
		appendPQExpBufferChar(aclbuf, ',');
	appendPQExpBuffer(aclbuf, "%s", keyword);
	if (subname)
		appendPQExpBuffer(aclbuf, "(%s)", subname);
}


static char *
copyAclUserName(PQExpBuffer output, char *input)
{
	resetPQExpBuffer(output);

	while (*input && *input != '=')
	{
		/*
		 * If user name isn't quoted, then just add it to the output buffer
		 */
		if (*input != '"')
			appendPQExpBufferChar(output, *input++);
		else
		{
			/* Otherwise, it's a quoted username */
			input++;
			/* Loop until we come across an unescaped quote */
			while (!(*input == '"' && *(input + 1) != '"'))
			{
				if (*input == '\0')
					return input;		/* really a syntax error... */

				/*
				 * Quoting convention is to escape " as "".  Keep this code in
				 * sync with putid() in backend's acl.c.
				 */
				if (*input == '"' && *(input + 1) == '"')
					input++;
				appendPQExpBufferChar(output, *input++);
			}
			input++;
		}
	}
	return input;
}

const char *
fmtId(const char *rawid)
{
	PQExpBuffer id_return = getLocalPQExpBuffer();

	const char *cp;
	bool		need_quotes = true;

	/*
	 * These checks need to match the identifier production in scan.l. Don't
	 * use islower() etc.
	 */
	if (!((rawid[0] >= 'a' && rawid[0] <= 'z') || rawid[0] == '_'))
		need_quotes = true;
	else
	{
		/* otherwise check the entire string */
		for (cp = rawid; *cp; cp++)
		{
			if (!((*cp >= 'a' && *cp <= 'z')
				  || (*cp >= '0' && *cp <= '9')
				  || (*cp == '_')))
			{
				need_quotes = true;
				break;
			}
		}
	}


	if (!need_quotes)
	{
		/* no quoting needed */
		appendPQExpBufferStr(id_return, rawid);
	}
	else
	{
		appendPQExpBufferChar(id_return, '\"');
		for (cp = rawid; *cp; cp++)
		{
			/*
			 * Did we find a double-quote in the string? Then make this a
			 * double double-quote per SQL99. Before, we put in a
			 * backslash/double-quote pair. - thomas 2000-08-05
			 */
			if (*cp == '\"')
				appendPQExpBufferChar(id_return, '\"');
			appendPQExpBufferChar(id_return, *cp);
		}
		appendPQExpBufferChar(id_return, '\"');
	}

	return id_return->data;
}

bool
parsePGArray(const char *atext, char ***itemarray, int *nitems)
{
	int			inputlen;
	char	  **items;
	char	   *strings;
	int			curitem;

	/*
	 * We expect input in the form of "{item,item,item}" where any item is
	 * either raw data, or surrounded by double quotes (in which case embedded
	 * characters including backslashes and quotes are backslashed).
	 *
	 * We build the result as an array of pointers followed by the actual
	 * string data, all in one malloc block for convenience of deallocation.
	 * The worst-case storage need is not more than one pointer and one
	 * character for each input character (consider "{,,,,,,,,,,}").
	 */
	*itemarray = NULL;
	*nitems = 0;
	inputlen = strlen(atext);
	if (inputlen < 2 || atext[0] != '{' || atext[inputlen - 1] != '}')
		return false;			/* bad input */
	items = (char **) malloc(inputlen * (sizeof(char *) + sizeof(char)));
	if (items == NULL)
		return false;			/* out of memory */
	*itemarray = items;
	strings = (char *) (items + inputlen);

	atext++;					/* advance over initial '{' */
	curitem = 0;
	while (*atext != '}')
	{
		if (*atext == '\0')
			return false;		/* premature end of string */
		items[curitem] = strings;
		while (*atext != '}' && *atext != ',')
		{
			if (*atext == '\0')
				return false;	/* premature end of string */
			if (*atext != '"')
				*strings++ = *atext++;	/* copy unquoted data */
			else
			{
				/* process quoted substring */
				atext++;
				while (*atext != '"')
				{
					if (*atext == '\0')
						return false;	/* premature end of string */
					if (*atext == '\\')
					{
						atext++;
						if (*atext == '\0')
							return false;		/* premature end of string */
					}
					*strings++ = *atext++;		/* copy quoted data */
				}
				atext++;
			}
		}
		*strings++ = '\0';
		if (*atext == ',')
			atext++;
		curitem++;
	}
	if (atext[1] != '\0')
		return false;			/* bogus syntax (embedded '}') */
	*nitems = curitem;
	return true;
}

static void
dumpDatabaseConfig(PGconn *conn, const char *dbname)
{
	PQExpBuffer buf = createPQExpBuffer();
	int			count = 1;

	for (;;)
	{
		PGresult   *res;

		if (server_version >= 90000)
			printfPQExpBuffer(buf, "SELECT setconfig[%d] FROM pg_db_role_setting WHERE "
							  "setrole = 0 AND setdatabase = (SELECT oid FROM pg_database WHERE datname = ", count);
		else
			printfPQExpBuffer(buf, "SELECT datconfig[%d] FROM pg_database WHERE datname = ", count);
		appendStringLiteralConn(buf, dbname, conn);

		if (server_version >= 90000)
			appendPQExpBuffer(buf, ")");

		appendPQExpBuffer(buf, ";");

		res = executeQuery(conn, buf->data, 0, NULL);
		if (PQntuples(res) == 1 &&
			!PQgetisnull(res, 0, 0))
		{
			makeAlterConfigCommand(conn, PQgetvalue(res, 0, 0),
								   "DATABASE", dbname, NULL, NULL);
			PQclear(res);
			count++;
		}
		else
		{
			PQclear(res);
			break;
		}
	}

	destroyPQExpBuffer(buf);
}

static PQExpBuffer
defaultGetLocalPQExpBuffer(void)
{
	static PQExpBuffer id_return = NULL;

	if (id_return)				/* first time through? */
	{
		/* same buffer, just wipe contents */
		resetPQExpBuffer(id_return);
	}
	else
	{
		/* new buffer */
		id_return = createPQExpBuffer();
	}

	return id_return;
}

static void
makeAlterConfigCommand(PGconn *conn, const char *arrayitem,
					   const char *type, const char *name,
					   const char *type2, const char *name2)
{
	char	   *pos;
	char	   *mine;
	PQExpBuffer buf;

	mine = pg_strdup(arrayitem);
	pos = strchr(mine, '=');
	if (pos == NULL)
	{
		free(mine);
		return;
	}

	buf = createPQExpBuffer();

	*pos = 0;
	appendPQExpBuffer(buf, "ALTER %s %s ", type, fmtId(name));
	if (type2 != NULL && name2 != NULL)
		appendPQExpBuffer(buf, "IN %s %s ", type2, fmtId(name2));
	appendPQExpBuffer(buf, "SET %s TO ", fmtId(mine));

	/*
	 * Some GUC variable names are 'LIST' type and hence must not be quoted.
	 */
	if (pg_strcasecmp(mine, "DateStyle") == 0
		|| pg_strcasecmp(mine, "search_path") == 0)
		appendPQExpBuffer(buf, "%s", pos + 1);
	else
		appendStringLiteralConn(buf, pos + 1, conn);
	appendPQExpBuffer(buf, ";\n");

	fprintf(OPF, "%s", buf->data);
	destroyPQExpBuffer(buf);
	free(mine);
}
