import os
import sys

from jinja2 import Environment, FileSystemLoader, StrictUndefined

from ldap2pg.defaults import make_well_known_acls
from ldap2pg.acl import process_definitions as process_acls
from ldap2pg import __version__


def slugify_filter(name):
    return name.replace('_', '-').strip('-')


def main(args=sys.argv[1:]):
    acls, groups, aliases = process_acls(make_well_known_acls())

    env = Environment(
        loader=FileSystemLoader(os.getcwd()),
        undefined=StrictUndefined,
        trim_blocks=True,
    )
    env.filters['slugify'] = slugify_filter
    template = env.get_template(args[0])
    values = dict(
        acls=acls,
        aliases=aliases,
        groups=groups,
        version=__version__,
    )
    print(template.render(**values))


if __name__ == '__main__':
    main()
