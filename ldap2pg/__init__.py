from pkg_resources import get_distribution

__dist__ = get_distribution('ldap2pg')
__version__ = __dist__.version
