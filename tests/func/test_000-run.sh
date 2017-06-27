#!/bin/bash -eux

# Fixtures
psql < dev-fixture.sql
ldapadd -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD -f dev-fixture.ldif

# Case dry run
DEBUG=1 DRY=1 ldap2pg
# Assert nothing is done
psql -c 'SELECT rolname FROM pg_roles;' | grep -q spurious

# Case real run
DEBUG=1 ldap2pg

# Assert spurious role is dropped
! psql -c 'SELECT rolname FROM pg_roles;' | grep -q spurious
test ${PIPESTATUS[0]} -eq 0

psql -c 'SELECT rolname FROM pg_roles WHERE rolsuper IS TRUE AND rolcanlogin IS TRUE;' | grep -q alice
psql -c 'SELECT rolname FROM pg_roles;' | grep -q bob
