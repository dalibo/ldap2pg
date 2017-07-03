from __future__ import unicode_literals

import logging

import psycopg2


logger = logging.getLogger(__name__)


class PSQL(object):
    def __init__(self, connstring=None):
        self.connstring = connstring or ''

    def __call__(self, dbname):
        connstring = self.connstring + " dbname=%s" % (dbname,)
        return PSQLSession(connstring.strip())


class PSQLSession(object):
    def __init__(self, connstring):
        self.connstring = connstring
        self.conn = None
        self.cursor = None

    def __enter__(self):
        logger.debug("Connecting to Postgres.")
        self.conn = psycopg2.connect(self.connstring)
        self.cursor = self.conn.cursor()
        return self

    def __exit__(self, *a):
        self.cursor.close()
        self.cursor = None
        self.conn.close()
        self.conn = None

    def __call__(self, query, *args):
        logger.debug("Doing: %s", query)
        self.cursor.execute(query, *args)
        self.conn.commit()
        return self.cursor

    @property
    def mogrify(self):
        return self.cursor.mogrify
