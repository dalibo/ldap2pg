#!/bin/bash -eux

# Trash all users
users=$(psql -tc "SELECT rolname FROM pg_roles WHERE rolname <> 'postgres' AND NOT rolname LIKE 'pg_%' ;"
)
for u in ${users} ; do
    dropuser --if-exists --echo $u
done

# Trash all entries
DN=$(ldapsearch -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD -b dc=ldap2pg,dc=local dn | grep '^dn: ' | sed '1d;s/dn: //' | tac)

for dn in ${DN} ; do
    if [[ $dn =~ admin ]] ; then
       continue
    fi

    ldapdelete -v -h $LDAP_HOST -D $LDAP_BIND -w $LDAP_PASSWORD $dn
done
