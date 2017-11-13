from __future__ import absolute_import, unicode_literals

from codecs import open
from collections import namedtuple
import logging
import os

from ldap import initialize as ldap_initialize, LDAPError
from ldap import (
    SCOPE_BASE,
    SCOPE_ONELEVEL,
    SCOPE_SUBORDINATE,
    SCOPE_SUBTREE,
)
from ldap.dn import str2dn as native_str2dn
from ldap import sasl

from .utils import decode_value, encode_value, PY2


logger = logging.getLogger(__name__)

__all__ = ['LDAPError']


SCOPES = {
    'base': SCOPE_BASE,
    'one': SCOPE_ONELEVEL,
    'sub': SCOPE_SUBTREE,
    'children': SCOPE_SUBORDINATE,
}

SCOPES_STR = {v: k for k, v in SCOPES.items()}


def parse_scope(raw):
    if raw in SCOPES_STR:
        return raw

    try:
        return SCOPES[raw]
    except KeyError:
        raise ValueError("Unknown scope %r" % (raw,))


if PY2:  # pragma: nocover_py3
    def str2dn(value):
        # Workaround buggy unicode managmenent in upstream python-ldap. This is
        # not necessary with pyldap on Python3.
        return decode_value(native_str2dn(value.encode('utf-8')))
else:
    str2dn = native_str2dn


class EncodedParamsCallable(object):  # pragma: nocover_py3
    # Wrap a callable not accepting unicode to encode all arguments.
    def __init__(self, callable_):
        self.callable_ = callable_

    def __call__(self, *a, **kw):
        a, kw = encode_value((a, kw))
        return decode_value(self.callable_(*a, **kw))


class UnicodeModeLDAPObject(object):  # pragma: nocover_py3
    # Simulate UnicodeMode from pyldap, on top of python-ldap. This is not a
    # Python2 issue but rather python-ldap not managing strings. Here we do it
    # for this.

    def __init__(self, wrapped):
        self.wrapped = wrapped

    def __getattr__(self, name):
        return EncodedParamsCallable(getattr(self.wrapped, name))


class LDAPLogger(object):
    def __init__(self, wrapped):
        self.wrapped = wrapped
        self.connect_opts = ''

    def __getattr__(self, name):
        return getattr(self.wrapped, name)

    def search_s(self, base, scope, filter, attributes):
        logger.debug(
            "Doing: ldapsearch%s -b %s -s %s '%s' %s",
            self.connect_opts,
            base, SCOPES_STR[scope], filter, ' '.join(attributes or []),
        )
        return self.wrapped.search_s(base, scope, filter, attributes)

    def simple_bind_s(self, binddn, password):
        self.connect_opts = ' -x'
        if password:
            self.connect_opts += ' -W'
        return self.wrapped.simple_bind_s(binddn, password)

    def sasl_interactive_bind_s(self, who, auth, *a, **kw):
        self.connect_opts = ' -Y %s' % (auth.mech.decode('ascii'),)
        if sasl.CB_AUTHNAME in auth.cb_value_dict:
            self.connect_opts += ' -U %s' % (
                auth.cb_value_dict[sasl.CB_AUTHNAME],)
        if sasl.CB_PASS in auth.cb_value_dict:
            self.connect_opts += ' -W'
        return self.wrapped.sasl_interactive_bind_s(who, auth, *a, **kw)


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
    logger.debug("Connecting to LDAP server %s.", options['URI'])
    l = ldap_initialize(options['URI'])
    if PY2:  # pragma: nocover_py3
        l = UnicodeModeLDAPObject(l)

    l = LDAPLogger(l)

    if options.get('USER'):
        logger.debug("Trying SASL DIGEST-MD5 auth.")
        auth = sasl.sasl({
            sasl.CB_AUTHNAME: options['USER'],
            sasl.CB_PASS: options['PASSWORD'],
        }, 'DIGEST-MD5')
        l.sasl_interactive_bind_s("", auth)
    else:
        logger.debug("Trying simple bind.")
        l.simple_bind_s(options['BINDDN'], options['PASSWORD'])

    return l


class Options(dict):
    def set_raw(self, option, raw):
        option = option.upper()
        try:
            parser = getattr(self, 'parse_' + option.lower())
        except AttributeError:
            logger.debug("Unknown option %s", option)
            return None
        else:
            value = parser(raw)
            self[option] = value
            return value

    def _parse_raw(self, value):
        return value

    parse_uri = _parse_raw
    parse_host = _parse_raw
    parse_port = int
    parse_binddn = _parse_raw
    parse_user = _parse_raw
    parse_password = _parse_raw


def gather_options(environ=None, **kw):
    options = Options(
        URI=None,
        HOST='',
        PORT=389,
        BINDDN=None,
        USER=None,
        PASSWORD='',
    )

    environ = environ or os.environ
    environ = {
        k[4:]: v.decode('utf-8') if hasattr(v, 'decode') else v
        for k, v in environ.items()
        if k.startswith('LDAP') and not k.startswith('LDAP2PG')
    }

    if 'NOINIT' in environ:
        logger.debug("LDAPNOINIT defined. Disabled ldap.conf loading.")
    else:
        for e in read_files(conf='/etc/ldap/ldap.conf', rc='ldaprc'):
            logger.debug('Read %s from %s.', e.option, e.filename)
            options.set_raw(e.option, e.value)
        for e in read_files(conf=options.get('CONF'), rc=options.get('RC')):
            logger.debug('Read %s from %s.', e.option, e.filename)
            options.set_raw(e.option, e.value)
        for option, value in environ.items():
            logger.debug('Read %s from env.', option)
            options.set_raw(option, value)

    options.update({
        k.upper(): v
        for k, v in kw.items()
        if k.upper() in options and v
    })

    if not options['URI']:
        options['URI'] = 'ldap://%(HOST)s:%(PORT)s' % options

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
            with open(candidate, 'r', encoding='utf-8') as fo:
                logger.debug('Found rcfile %s.', candidate)
                for entry in parserc(fo):
                    yield entry
        except (IOError, OSError) as e:
            logger.debug("Ignoring: %s", e)


RCEntry = namedtuple('RCEntry', ('filename', 'lineno', 'option', 'value'))


def parserc(fo):
    filename = getattr(fo, 'name', '<stdin>')

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
