#!/bin/bash -eux

# Fixtures
psql < dev-fixture.sql
ldapadd -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD -f dev-fixture.ldif

list_superusers() {
    psql -tc "SELECT rolname FROM pg_roles WHERE rolsuper IS TRUE;"
}

list_members() {
    psql -t <<EOSQL
SELECT m.rolname FROM pg_roles AS m
JOIN pg_auth_members a ON a.member = m.oid
JOIN pg_roles AS r ON r.oid = a.roleid
ORDER BY 1;
EOSQL
}

list_roles() {
    psql -tc "SELECT rolname FROM pg_roles;"
}

# Case dry run
DEBUG=1 DRY=1 ldap2pg
# Assert nothing is done
list_users | grep -q spurious

# Case real run
DEBUG=1 DRY=0 ldap2pg

# Assert spurious role is dropped
! list_users | grep -q spurious
test ${PIPESTATUS[0]} -eq 0

list_superusers | grep -q alice
list_roles | grep -q bob
list_members app0 | grep -q foo
list_members app1 | grep -q bar
