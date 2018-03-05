import sys
from setuptools import setup


PY2 = sys.version_info < (3,)
PY26 = sys.version_info < (2, 7)

install_requires = [
    'psycopg2',
    'pyyaml',
]

if PY2:
    install_requires.append('python-ldap')
    if PY26:
        install_requires.extend([
            'argparse',
            'logutils',
        ])
else:
    # python-ldap does not support Python3
    install_requires.append('pyldap')

setup(
    name='ldap2pg',
    version='4.7',
    description='Synchronize PostgreSQL roles and ACLs from LDAP',
    url='https://github.com/dalibo/ldap2pg',
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
