#!/bin/bash -eux

ldap2pg --help
ldap2pg -?
ldap2pg -vn

export LDAP2PG_CONFIG=my-test-ldap2pg.yml
rm -f $LDAP2PG_CONFIG
# File does not exists -> no syncmap
! ldap2pg

cat > $LDAP2PG_CONFIG <<EOYAML
ldap:
  host: ${LDAP_HOST}
  port: ${LDAP_PORT-389}
  bind: ${LDAP_BIND}
  password: ${LDAP_PASSWORD}

sync_map:
- ldap:
    base: cn=dba,ou=groups,dc=ldap2pg,dc=local
    filter: "(objectClass=groupOfNames)"
    attribute: member
  role:
    name_attribute: member.cn
    options: LOGIN SUPERUSER NOBYPASSRLS
EOYAML

var_bl=(${!LDAP_*})
sandbox="env ${var_bl[@]/#/--unset }"
# File is world readable
! $sandbox ldap2pg

chmod 0600 ${LDAP2PG_CONFIG}

# Now it's ok :)
$sandbox ldap2pg
