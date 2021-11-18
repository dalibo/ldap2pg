from __future__ import absolute_import, unicode_literals

from codecs import open
from collections import namedtuple
import logging
import os
try:
    from shlex import quote as shquote
except ImportError:  # pragma: nocover_py3
    def shquote(x):
        return "'%s'" % x


import ldap

# On CentOS 6, python-ldap does not manage SCOPE_SUBORDINATE
try:
    from ldap import SCOPE_SUBORDINATE
except ImportError:  # pragma: nocover
    SCOPE_SUBORDINATE = None

from ldap.dn import str2dn as native_str2dn
from ldap import sasl

from .format import FormatVars, FormatValue
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

DN_COMPONENTS = ('cn', 'l', 'st', 'o', 'ou', 'c', 'street', 'dc', 'uid')


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


class MissingAttributeError(KeyError):
    def __str__(self):
        return "Missing attribute: %s." % (self.args[0])


class LDAPEntry(object):
    __slot__ = (
        'dn',
        'attributes',
        'children',
        '__dict__',
    )

    def __init__(self, dn, attributes=None, children=None):
        self.dn = dn
        self.attributes = attributes or {}
        self.children = children or {}

    def __eq__(self, other):
        return (
            self.dn == other.dn
            and self.attributes == other.attributes
            and self.children == other.children
        )

    def __repr__(self):
        return '<%s %s>' % (
            self.__class__.__name__,
            self.dn,
        )

    def __getitem__(self, key):
        # Generate all values from entry for accessor attribute. Attribute can
        # be a single attribute name, a path to a RDN in a distinguished name,
        # or an attribute of a join (aka subquery).

        if "dn" == key:
            yield self.dn
            return

        path = key.lower().split('.')

        # First level access.
        try:
            values = self.attributes[path[0]]
            path = path[1:]
        except KeyError:
            # Fallback DN components in DN, just like having `dn.XX`.
            if path[0] in DN_COMPONENTS:
                values = [self.dn]
            elif "dn" == path[0]:
                values = [self.dn]
                path = path[1:]
            else:
                raise MissingAttributeError(path[0])

        if not path:
            for value in values:
                yield value
        elif path[0] in DN_COMPONENTS:
            for value in values:
                raw_dn = value
                try:
                    dn = str2dn(value)
                except ValueError:
                    msg = "Can't parse DN from attribute %s=%s." % (
                        key, value)
                    raise ValueError(msg)
                value = dict()
                for (type_, name, _), in dn:
                    value.setdefault(type_.lower(), name)
                try:
                    yield value[path[0]]
                except KeyError:
                    yield RDNError("Unknown RDN %s." % (path[0],), raw_dn)
        else:
            raise MissingAttributeError(path[0])

    def build_format_vars(self, map_, processor=None):
        # Builds a dicts of values from self corresponding to the request
        # described in map_.

        if processor is None:
            def processor(x):
                return x

        vars_ = FormatVars(map_)
        for objname, attributes in map_.items():
            prefix = ''
            if objname in ("__self__", "dn"):
                entries = [self]
            else:
                try:
                    entries = self.children[objname]
                except KeyError:
                    # Accessing a RDN of a foreign-key like {member.cn}. Fake
                    # sub-query entries for each DN as returned by
                    # __self__.member.
                    entries = [
                        LDAPEntry(dn) for dn in processor(self[objname])
                    ]

            vars_[objname] = []
            for entry in entries:
                entry_vars = {}

                for attr in attributes:
                    fullattr = prefix + attr
                    if attr.startswith("dn."):
                        dn = entry_vars.get("dn")
                        if not isinstance(dn, dict):
                            dn = dict(dn=FormatValue(entry.dn))
                            entry_vars["dn"] = [dn]
                        dn[attr[3:]] = FormatValue(next(entry[fullattr]))
                    else:
                        entry_vars[attr] = [
                            FormatValue(v) for v in processor(entry[fullattr])
                        ]

                entry_vars.setdefault("dn", [FormatValue(entry.dn)])
                vars_[objname].append(entry_vars)

        return vars_


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
            "Doing: ldapsearch%s -b %s -s %s %s %s",
            self.connect_opts,
            base, SCOPES_STR[scope], shquote(filter),
            ' '.join(attributes or []),
        )
        with self.timer:
            return self.wrapped.search_s(base, scope, filter, attributes)

    def simple_bind_s(self, binddn, password):
        self.connect_opts = ' -x'
        if binddn:
            self.connect_opts += ' -D %s' % (shquote(binddn),)
        if password:
            self.connect_opts += ' -W'
        self.log_connect()
        return self.wrapped.simple_bind_s(binddn, password)

    def sasl_interactive_bind_s(self, who, auth, *a, **kw):
        self.connect_opts = ' -Y %s' % (auth.mech.decode('ascii'),)
        if sasl.CB_AUTHNAME in auth.cb_value_dict:
            self.connect_opts += ' -U %s' % (
                shquote(auth.cb_value_dict[sasl.CB_AUTHNAME]),)
        if sasl.CB_PASS in auth.cb_value_dict:
            self.connect_opts += ' -W'
        self.log_connect()
        return self.wrapped.sasl_interactive_bind_s(who, auth, *a, **kw)

    def log_connect(self):
        logger.debug("Authenticating: ldapwhoami%s", self.connect_opts)


def connect(**kw):
    # Sources order, see ldap.conf(3)
    #   variable     $LDAPNOINIT, and if that is not set:
    #   system file  /etc/ldap/ldap.conf, /etc/openldap/ldap.conf
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
    conn.set_option(ldap.OPT_NETWORK_TIMEOUT, 120)

    if options.get('STARTTLS'):
        logger.debug("Sending STARTTLS.")
        conn.set_option(ldap.OPT_X_TLS_NEWCTX, 0)
        conn.start_tls_s()

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
    parse_starttls = _parse_bool


def gather_options(environ=None, **kw):
    # This is the main point for LDAP configuration marshall. kw is the ldap
    # stanza in ldap2pg.yml.
    #
    # ldap2pg handles a subset of openldap libldap parameters. These parameters
    # are declared in options variable with their default value as for ldap2pg.
    # These parameters are used in ldap2pg client-side logic.
    #
    # Value for these parameters are searching in ldap.conf files, environment
    # variables and YAML file.
    default_conffiles = [
        '/etc/openldap/ldap.conf',
        '/etc/ldap/ldap.conf',
    ]
    options = Options(
        URI='',
        HOST='',
        PORT=389,
        BINDDN='',
        USER=None,
        PASSWORD='',
        SASL_MECH=None,
        # This is an extension to ldap.conf. Equivalent of -Z CLI options of
        # ldap-utils.
        STARTTLS=False,
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
        for e in read_files(conf=default_conffiles, rc='ldaprc'):
            logger.debug('Read %s from %s.', e.option, e.filename)
            options.set_raw(e.option, e.value)
        customconf = environ.pop('CONF', options.get('CONF'))
        customrc = environ.pop('RC', options.get('RC'))
        for e in read_files(conf=customconf, rc=customrc):
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
    if isinstance(conf, list):
        candidates.extend(conf)
    elif isinstance(conf, str):
        candidates.append(conf)
    if rc:
        candidates.extend(['~/%s' % rc, '~/.%s' % rc, './%s' % rc])
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

        try:
            option, value = line.split(None, 1)
        except ValueError:
            raise UserError(
                "Bad syntax in %s at line %s: %s" % (filename, lineno, line))

        yield RCEntry(
            filename=filename,
            lineno=lineno+1,
            option=option,
            value=value,
        )
