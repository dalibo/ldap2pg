import os
import sys

from jinja2 import Environment, FileSystemLoader, StrictUndefined

from ldap2pg.defaults import make_well_known_privileges
from ldap2pg.privilege import process_definitions as process_privileges
from ldap2pg import __version__


def escape_markdown(string):
    return string.replace('_', r'\_')


def slugify_filter(name):
    return name.replace('_', '-').strip('-')


def main(args=sys.argv[1:]):
    privileges, groups, aliases = process_privileges(
        make_well_known_privileges())

    env = Environment(
        loader=FileSystemLoader(os.getcwd()),
        undefined=StrictUndefined,
        trim_blocks=True,
    )
    env.filters['slugify'] = slugify_filter
    env.filters['escape_markdown'] = escape_markdown
    template = env.get_template(args[0])
    values = dict(
        privileges=privileges,
        aliases=aliases,
        groups=groups,
        version=__version__,
    )
    print(template.render(**values))


if __name__ == '__main__':
    main()
