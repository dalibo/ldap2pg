#!/bin/bash

set -x
# Declare LDAP service
samba-tool spn add ldap/samba.ldap2pg.docker 'SAMBA$'
samba-tool spn add ldap/localhost 'SAMBA$'
samba-tool spn add ldap/localhost.localdomain 'SAMBA$'
samba-tool spn list 'SAMBA$'

# Export keytab for kinit and kerberos clients.
samba-tool domain exportkeytab /test/samba.keytab --principal=Administrator
chown -v "$(stat -c %u:%g "${BASH_SOURCE[0]}")" /test/samba.keytab
