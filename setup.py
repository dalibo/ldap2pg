import sys
from setuptools import setup


PY3 = sys.version_info > (3,)


setup(
    install_requires=[
        'psycopg2',
        'pyldap' if PY3 else 'python-ldap',
        'pyyaml',
        'six',
    ],
    # Se setup.cfg for metadata and other parameters.
)
