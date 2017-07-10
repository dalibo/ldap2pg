from __future__ import absolute_import

from collections import namedtuple
import logging
import os

import ldap3


logger = logging.getLogger(__name__)


def connect(**kw):
    # Sources order, see ldap.conf(3)
    #   variable     $LDAPNOINIT, and if that is not set:
    #   system file  /etc/ldap/ldap.conf,
    #   user files   $HOME/ldaprc,  $HOME/.ldaprc,  ./ldaprc,
    #   system file  $LDAPCONF,
    #   user files   $HOME/$LDAPRC, $HOME/.$LDAPRC, ./$LDAPRC,
    #   user files   <ldap2pg.yml>...
    #   variables    $LDAP<uppercase option name>.
    #
    # Extra variable LDAPPASSWORD is supported.

    options = gather_options(**kw)
    logger.debug(
        "Connecting to LDAP server %s:%s.",
        options['HOST'], options['PORT'],
    )
    server = ldap3.Server(options['HOST'], options['PORT'])
    return ldap3.Connection(
        server, options['BINDDN'], options['PASSWORD'], auto_bind=True,
    )


class Options(dict):
    def set_raw(self, option, raw):
        option = option.upper()
        try:
            parser = getattr(self, 'parse_' + option.lower())
        except AttributeError:
            logger.debug("Unkown option %s", option)
            return None
        else:
            value = parser(raw)
            self[option] = value
            return value

    def _parse_raw(self, value):
        return value

    parse_host = _parse_raw
    parse_port = int
    parse_binddn = _parse_raw
    parse_base = _parse_raw
    parse_password = _parse_raw


def gather_options(environ=None, **kw):
    options = Options(
        HOST='',
        PORT=389,
        BINDDN=None,
        PASSWORD=None,
    )

    environ = environ or os.environ
    environ = {
        k[4:]: v
        for k, v in environ.items()
        if k.startswith('LDAP') and not k.startswith('LDAP2PG')
    }

    if 'NOINIT' in environ:
        logger.debug("LDAPNOINIT defined. Disabled ldap.conf loading.")
    else:
        for e in read_files(conf='/etc/ldap/ldap.conf', rc='ldaprc'):
            options.set_raw(e.option, e.value)
        for e in read_files(conf=options.get('CONF'), rc=options.get('RC')):
            options.set_raw(e.option, e.value)
        for option, value in environ.items():
            options.set_raw(option, value)

    options.update({
        k.upper(): v
        for k, v in kw.items()
        if k.upper() in options and v
    })

    return options


def read_files(conf, rc):
    candidates = []
    if conf:
        candidates.append(conf)
    if rc:
        candidates.extend(['~/%s' % rc, '~/.%s' % rc, rc])

    for candidate in candidates:
        candidate = os.path.expanduser(candidate)
        try:
            with open(candidate, 'r') as fo:
                logger.debug('Found rcfile %s.', candidate)
                for entry in parserc(fo):
                    yield entry
        except (IOError, OSError) as e:
            logger.debug("Ignoring: %s", e)


RCEntry = namedtuple('RCEntry', ('filename', 'lineno', 'option', 'value'))


def parserc(fo):
    filename = getattr(fo, 'filename', '<stdin>')

    for lineno, line in enumerate(fo):
        line = line.strip()
        if not line:
            continue

        if line.startswith('#'):
            continue

        option, value = line.split(None, 1)
        yield RCEntry(
            filename=filename,
            lineno=lineno+1,
            option=option,
            value=value,
        )
