from __future__ import unicode_literals

import ctypes
import logging
import re
from contextlib import closing

from psycopg2 import connect, __version__ as psycopg2_version
import psycopg2.extensions

from .utils import (
    AllDatabases,
    UserError,
    ensure_unicode,
    lower1,
    urlparse,
    urlunparse,
)


# Add a new log level change, between INFO and WARNING.

class ChangeLogger(logging.getLoggerClass()):
    def change(self, msg, *args, **kwargs):
        if self.isEnabledFor(logging.CHANGE):
            self._log(logging.CHANGE, msg, args, **kwargs)


logging.CHANGE = logging.INFO + 5
logging.addLevelName(logging.CHANGE, 'CHANGE')
# Use ChangeLogger class only in this module.
logging.setLoggerClass(ChangeLogger)
logger = logging.getLogger(__name__)
logging.setLoggerClass(logging.Logger)
logger.change  # Raises AttributeError if logger is not ChangeLogger.

psycopg2.extensions.register_type(psycopg2.extensions.UNICODE)
psycopg2.extensions.register_type(psycopg2.extensions.UNICODEARRAY)


class Pooler(object):
    def __init__(self, connstring, size=256, dry=False):
        self.connstring = connstring
        self.size = size
        self.connections = {}

    def getconn(self, dbname=None):
        try:
            return self.connections[dbname]
        except KeyError:
            pass

        if len(self) >= self.size:
            msg = (
                "Database limit exceeded.\n"
                "ldap2pg doesn't support cluster with more than %d databases."
            ) % (self.size)
            raise UserError(msg)

        logger.debug("Opening connection to %s.", dbname or 'libpq default')

        connstring = self.connstring
        kw = {}
        if psycopg2_version > '2.5':
            # application_name is not available on CentOS 6. Use psycopg2
            # version because libpq.PQlibVersion is not available with Python
            # 2.6.
            kw['application_name'] = 'ldap2pg'
            kw['dbname'] = dbname
        elif dbname:
            connstring = inject_database_in_connstring(self.connstring, dbname)

        self.connections[dbname] = conn = connect(
            connstring, connection_factory=FactoryConnection, **kw
        )
        if psycopg2_version > '2.4':
            conn.set_session(autocommit=True)
        return conn

    def putconn(self, dbname=None):
        conn = self.connections.pop(dbname, None)
        if conn:
            logger.debug(
                "Closing connection to %s.", dbname or 'libpq default')
            conn.close()

    def __enter__(self):
        return self

    def __exit__(self, *a):
        for name in list(self.connections.keys()):
            self.putconn(name)

    def __iter__(self):
        for dbname, conn in self.pool.items():
            yield dbname, conn

    def __len__(self):
        return len(self.connections)

    _dbname_re = re.compile("dbname *= *'?[^ ]*'?")


class FactoryConnection(psycopg2.extensions.connection):  # pragma: nocover
    def cursor(self, *a, **kw):
        row_factory = kw.pop('row_factory', None)
        kw['cursor_factory'] = FactoryCursor.make_factory(
            row_factory=row_factory)
        return super(FactoryConnection, self).cursor(*a, **kw)

    def execute(self, sql, *args):
        # Use closing for psycopg 2.0.
        with closing(self.cursor()) as cur:
            cur.execute(sql, *args)

        # # Autocommit for psycopg 2.0 on CentOS 6.
        if not getattr(self, 'autocommit', False):
            self.commit()

    def query(self, row_factory, sql, *args):
        with closing(self.cursor(row_factory=row_factory)) as cur:
            cur.execute(sql, *args)
            for row in cur.fetchall():
                yield row

    def queryone(self, row_factory, sql, *args):
        with closing(self.cursor(row_factory=row_factory)) as cur:
            cur.execute(sql, *args)
            return cur.fetchone()

    def scalar(self, sql, *args):
        return self.queryone(scalar, sql, *args)

    def mogrify(self, qry, *a, **kw):
        with closing(self.cursor()) as cur:
            sql = cur.mogrify(qry.encode('utf-8'), *a, **kw)
        target_encoding = psycopg2.extensions.encodings[self.encoding]
        try:
            return sql.decode(target_encoding)
        except UnicodeDecodeError:
            raise UserError(
                "Can't encode query to database encoding %s."
                % self.conn.encoding)


class FactoryCursor(psycopg2.extensions.cursor):  # pragma: nocover
    # Implement row_factory for psycopg2.

    @classmethod
    def make_factory(cls, row_factory=None):
        # Build a cursor_factory for psycopg2 connection.
        def factory(*a, **kw):
            kw['row_factory'] = row_factory
            return cls(*a, **kw)
        return factory

    def __init__(self, conn, name=None, row_factory=None):
        super(FactoryCursor, self).__init__(conn)
        if not row_factory:
            def row_factory(*a):
                return a
        self._row_factory = row_factory

    def execute(self, query, *a, **kw):
        if a:
            raise Exception("XXX MOGRIFY")
        logger.debug("Doing:\n%s", query.strip())
        return super(FactoryCursor, self).execute(query, *a, **kw)

    def fetchone(self):
        return self._row_factory(*super(FactoryCursor, self).fetchone())

    def fetchmany(self, size=None):
        for row in super(FactoryCursor, self).fetchmany(size):
            yield self._row_factory(*row)

    def fetchall(self):
        for row in super(FactoryCursor, self).fetchall():
            yield self._row_factory(*row)


def scalar(col0, *_):
    # Row factory for scalar.
    return col0


class Query(object):
    # Represent a query with log message and target database.

    ALL_DATABASES = AllDatabases()

    def __init__(self, message, dbname, *args):
        self.message = message
        self.dbname = dbname
        self.args = args

    def __repr__(self):
        return "<%s on %s: %r>" % (
            self.__class__.__name__,
            self.dbname,
            self.args[0][:50] + '...',
        )

    def __str__(self):
        return self.message

    def expand(self, databases):
        if self.dbname is self.ALL_DATABASES:
            for dbname in databases:
                yield Query(
                    self.message % dict(dbname=dbname),
                    dbname,
                    *self.args
                )
        else:
            yield self


def expand_queries(queries, databases):
    for query in queries:
        for single_query in query.expand(databases):
            yield single_query


def execute_queries(pool, queries, timer, dry=False):
    count = 0
    for query in queries:
        conn = pool.getconn(query.dbname)
        count += 1
        if dry:
            logger.change('Would ' + lower1(query.message))
        else:
            logger.change(query.message)

        sql = conn.mogrify(*query.args)
        if dry:
            logger.debug("Would execute: %s", sql)
            continue

        try:
            with timer:
                conn.execute(sql)
        except Exception as e:
            fmt = "Error while executing SQL query:\n%s"
            raise UserError(fmt % ensure_unicode(e))

    return count


def libpq_version():
    # Search libpq version bound to this process.

    try:
        return psycopg2.__libpq_version__
    except AttributeError:
        # Search for libpq.so path in loaded libraries.
        with open('/proc/self/maps') as fo:
            for line in fo:
                values = line.split()
                path = values[-1]
                if '/libpq' in path:
                    break
            else:  # pragma: nocover
                raise Exception("libpq.so not loaded")

        libpq = ctypes.cdll.LoadLibrary(path)
        return libpq.PQlibVersion()


_dbname_re = re.compile("dbname *= *'?[^ ]*'?")


def inject_database_in_connstring(connstring, dbname):

    if dbname is None:
        return connstring

    if (connstring.startswith('postgres://') or
            connstring.startswith('postgresql://')):
        pr = list(urlparse(connstring))
        pr[2] = dbname
        return urlunparse(pr)
    else:
        connstring = _dbname_re.sub('', connstring)
        return connstring + " dbname=%s" % (dbname,)
