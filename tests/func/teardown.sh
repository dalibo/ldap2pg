#!/bin/bash -eux

# Trash all Postgres roles and memberships
psql -t <<EOSQL
DELETE FROM pg_catalog.pg_auth_members;
DELETE FROM pg_catalog.pg_authid WHERE rolname != 'postgres' AND rolname NOT LIKE 'pg_%';
EOSQL

# Trash all LDAP entries
DN=$(ldapsearch -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD -b dc=ldap2pg,dc=local dn | grep '^dn: ' | sed '1d;s/dn: //' | tac)

for dn in ${DN} ; do
    if [[ $dn =~ admin ]] ; then
       continue
    fi

    ldapdelete -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD $dn
done
