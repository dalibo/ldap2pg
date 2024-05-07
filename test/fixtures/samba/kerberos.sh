#!/bin/bash

set -x
samba-tool spn add ldap/samba1.ldap2pg.docker SAMBA1$
samba-tool spn add ldap/localhost SAMBA1$
samba-tool spn add ldap/localhost.localdomain SAMBA1$

# Get Gateway (field 3) from default route (destination is 0.0.0.0).
gateway_hex="$(grep -E '^\w+\s+00000000' /proc/net/route | cut -f 3)"
gateway_bytes=( # IP is little endian.
	$((16#${gateway_hex:6:2}))
	$((16#${gateway_hex:4:2}))
	$((16#${gateway_hex:2:2}))
	$((16#${gateway_hex:0:2}))
)
printf -v gateway "%d.%d.%d.%d" "${gateway_bytes[@]}"

samba-tool spn add "ldap/$gateway" SAMBA1$
samba-tool spn list SAMBA1$
samba-tool domain exportkeytab /test/samba.keytab --principal=Administrator
chown -v "$(stat -c %u:%g "${BASH_SOURCE[0]}")" /test/samba.keytab
