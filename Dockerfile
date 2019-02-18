FROM python:3.7-slim

RUN  apt-get update && apt-get install -y libldap2-dev libsasl2-dev python3-pip && pip install psycopg2==2.7.3.2 ldap2pg # installing 
#Installing psycopg2==2.7.3.2 avoiding warning on tool run:
#The psycopg2 wheel package will be renamed from release 2.8; in order to keep installing from binary please use "pip install psycopg2-binary" instead. For details see: <http://initd.org/psycopg/docs/install.html#binary-install-from-pypi>.
