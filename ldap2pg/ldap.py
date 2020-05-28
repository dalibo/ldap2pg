from __future__ import absolute_import, unicode_literals

from codecs import open
from collections import namedtuple
import logging
import os

import ldap

# On CentOS 6, python-ldap does not manage SCOPE_SUBORDINATE
try:
    from ldap import SCOPE_SUBORDINATE
except ImportError:  # pragma: nocover
    SCOPE_SUBORDINATE = None

from ldap.dn import str2dn as native_str2dn
from ldap import sasl

from .utils import decode_value, encode_value, PY2, uniq
from .utils import Timer
from .utils import UserError


logger = logging.getLogger(__name__)

LDAPError = ldap.LDAPError


SCOPES = {
    'base': ldap.SCOPE_BASE,
    'one': ldap.SCOPE_ONELEVEL,
    'sub': ldap.SCOPE_SUBTREE,
}

if SCOPE_SUBORDINATE:
    SCOPES['children'] = SCOPE_SUBORDINATE

SCOPES_STR = dict((v, k) for k, v in SCOPES.items())

DN_COMPONENTS = ('dn', 'cn', 'l', 'st', 'o', 'ou', 'c', 'street', 'dc', 'uid')


def parse_scope(raw):
    if raw in SCOPES_STR:
        return raw

    try:
        return SCOPES[raw]
    except KeyError:
        raise ValueError("Unknown scope %r" % (raw,))


def str2dn(value):
    try:
        if PY2:  # pragma: nocover_py3
            # Workaround buggy unicode managmenent in python-ldap on Python2.
            # This is not necessary on Python3.
            value = decode_value(native_str2dn(value.encode('utf-8')))
        else:  # pragma: nocover_py2
            value = native_str2dn(value)
    except ldap.DECODING_ERROR:
        raise ValueError("Can't parse DN '%s'" % (value,))

    return [
        [(k.lower(), v, _) for k, v, _ in t]
        for t in value
    ]


class RDNError(NameError):
    # Raised when an unexpected DN is reached.
    def __init__(self, message=None, dn=None):
        super(RDNError, self).__init__(message)
        self.dn = dn


def get_attribute(entry, attribute):
    # Generate all values from entry for accessor attribute. Attribute can be a
    # single attribute name, a path to a RDN in a distinguished name, or an
    # attribute of a join (aka subquery).
    _, attributes, joins = entry
    path = attribute.lower().split('.')
    try:
        values = attributes[path[0]]
    except KeyError:
        raise ValueError("Unknown attribute %r" % (path[0],))

    attribute = path[0]
    path = path[1:]
    if not path:
        for value in values:
            yield value
    elif path[0] in DN_COMPONENTS:
        for value in values:
            raw_dn = value
            try:
                dn = str2dn(value)
            except ValueError:
                msg = "Can't parse DN from attribute %s=%s" % (
                    attribute, value)
                raise ValueError(msg)
            value = dict()
            for (type_, name, _), in dn:
                value.setdefault(type_.lower(), name)
            try:
                yield value[path[0]]
            except KeyError:
                yield RDNError("Unknown RDN %s" % (path[0],), raw_dn)
    else:
        try:
            joined_entries = joins[attribute]
        except KeyError:
            msg = "Missing join result for %s" % (attribute,)
            raise ValueError(msg)

        for joined_entry in joined_entries:
            for value in get_attribute(joined_entry, '.'.join(path)):
                yield value


def lower_attributes(entry):
    dn, attributes = entry
    return dn, dict([
        (k.lower(), v)
        for k, v in attributes.items()
    ])


class EncodedParamsCallable(object):  # pragma: nocover_py3
    # Wrap a callable not accepting unicode to encode all arguments.
    def __init__(self, callable_):
        self.callable_ = callable_

    def __call__(self, *a, **kw):
        a, kw = encode_value((a, kw))
        return decode_value(self.callable_(*a, **kw))


class UnicodeModeLDAPObject(object):  # pragma: nocover_py3
    # Simulate UnicodeMode from Python3, on top of python-ldap. This is not a
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
        self.timer = Timer()

    def __getattr__(self, name):
        return getattr(self.wrapped, name)

    def search_s(self, base, scope, filter, attributes):
        logger.debug(
            "Doing: ldapsearch%s -b %s -s %s '%s' %s",
            self.connect_opts,
            base, SCOPES_STR[scope], filter, ' '.join(attributes or []),
        )
        with self.timer:
            return self.wrapped.search_s(base, scope, filter, attributes)

    def simple_bind_s(self, binddn, password):
        self.connect_opts = ' -x'
        if binddn:
            self.connect_opts += ' -D %s' % (binddn,)
        if password:
            self.connect_opts += ' -W'
        self.log_connect()
        return self.wrapped.simple_bind_s(binddn, password)

    def sasl_interactive_bind_s(self, who, auth, *a, **kw):
        self.connect_opts = ' -Y %s' % (auth.mech.decode('ascii'),)
        if sasl.CB_AUTHNAME in auth.cb_value_dict:
            self.connect_opts += ' -U %s' % (
                auth.cb_value_dict[sasl.CB_AUTHNAME],)
        if sasl.CB_PASS in auth.cb_value_dict:
            self.connect_opts += ' -W'
        self.log_connect()
        return self.wrapped.sasl_interactive_bind_s(who, auth, *a, **kw)

    def log_connect(self):
        logger.debug("Doing: ldapwhoami%s", self.connect_opts)


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
    conn = ldap.initialize(options['URI'])
    if PY2:  # pragma: nocover_py3
        conn = UnicodeModeLDAPObject(conn)

    conn = LDAPLogger(conn)
    # Don't follow referrals by default. This is the behaviour of ldapsearch
    # and friends. Following referrals leads to strange errors with Active
    # directory. REFERRALS can still be activated through ldaprc, env var and
    # even YAML. See https://github.com/dalibo/ldap2pg/issues/228 .
    conn.set_option(ldap.OPT_REFERRALS, options.get('REFERRALS', False))

    if not options.get('SASL_MECH'):
        logger.debug("Trying simple bind.")
        conn.simple_bind_s(options['BINDDN'], options['PASSWORD'])
    else:
        logger.debug("Trying SASL %s auth.", options['SASL_MECH'])
        mech = options['SASL_MECH']
        if 'DIGEST-MD5' == mech:
            auth = sasl.sasl({
                sasl.CB_AUTHNAME: options['USER'],
                sasl.CB_PASS: options['PASSWORD'],
            }, mech)
        elif 'GSSAPI' == mech:
            auth = sasl.gssapi(options.get('SASL_AUTHZID'))
        else:
            raise UserError("Unmanaged SASL mech %s.", mech)

        conn.sasl_interactive_bind_s("", auth)

    return conn


class Options(dict):
    def set_raw(self, option, raw):
        option = option.upper()
        try:
            parser = getattr(self, 'parse_' + option.lower())
        except AttributeError:
            return None
        else:
            value = parser(raw)
            self[option] = value
            return value

    def _parse_raw(self, value):
        return value

    def _parse_bool(self, value):
        return value not in (False, 'false', 'no', 'off')

    parse_uri = _parse_raw
    parse_host = _parse_raw
    parse_port = int
    parse_binddn = _parse_raw
    parse_user = _parse_raw
    parse_password = _parse_raw
    parse_sasl_mech = _parse_raw
    parse_referrals = _parse_bool


def gather_options(environ=None, **kw):
    options = Options(
        URI='',
        HOST='',
        PORT=389,
        BINDDN='',
        USER=None,
        PASSWORD='',
        SASL_MECH=None,
        REFERRALS=False,
    )

    environ = environ or os.environ
    environ = dict([
        (k[4:], v.decode('utf-8') if hasattr(v, 'decode') else v)
        for k, v in environ.items()
        if k.startswith('LDAP') and not k.startswith('LDAP_')
    ])

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

    for k, v in kw.items():
        if v is None:
            continue
        k = k.upper()
        if k not in options:
            continue
        logger.debug('Read %s from YAML.', k)
        options.set_raw(k, v)

    if not options['URI']:
        options['URI'] = 'ldap://%(HOST)s:%(PORT)s' % options

    if options.get('USER'):
        options['SASL_MECH'] = 'DIGEST-MD5'

    return options


def read_files(conf, rc):
    candidates = []
    if conf:
        candidates.append(conf)
    if rc:
        candidates.extend(['~/%s' % rc, '~/.%s' % rc, rc])
    candidates = uniq(map(
        lambda p: os.path.realpath(os.path.expanduser(p)),
        candidates,
    ))

    for candidate in candidates:
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
