import sys
from setuptools import setup


PY3 = sys.version_info > (3,)


setup(
    name='ldap2pg',
    version='3.2',
    description='Synchronize PostgreSQL roles from LDAP',
    url='https://github.com/dalibo/ldap2pg',
    author='Dalibo',
    author_email='contact@dalibo.com',
    license='PostgreSQL',
    install_requires=[
        'psycopg2',
        'pyldap' if PY3 else 'python-ldap',
        'pyyaml',
    ],
    packages=['ldap2pg'],
    entry_points={
        'console_scripts': ['ldap2pg = ldap2pg.script:main']
    }
    # See setup.cfg for other metadata and parameters.
)
