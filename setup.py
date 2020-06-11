import sys
from setuptools import setup


PY26 = sys.version_info < (2, 7)

install_requires = [
    'python-ldap',
    'pyyaml',
]

if PY26:
    install_requires.extend([
        'argparse',
        'logutils',
    ])

setup(
    name='ldap2pg',
    version='5.4',
    description='Manage PostgreSQL roles and privileges from YAML or LDAP',
    url='https://labs.dalibo.com/ldap2pg',
    project_urls={
        "Documentation": "https://ldap2pg.rtfd.io/",
        "Tracker": "https://github.com/dalibo/ldap2pg/issues",
        "Sources": "https://github.com/dalibo/ldap2pg",
    },
    author='Dalibo',
    author_email='contact@dalibo.com',
    license='PostgreSQL',
    install_requires=install_requires,
    packages=['ldap2pg'],
    entry_points={
        'console_scripts': ['ldap2pg = ldap2pg.script:main']
    }
    # See setup.cfg for other metadata and parameters.
)
