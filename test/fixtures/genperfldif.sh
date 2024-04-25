#!/bin/bash

set -eu

cat <<-EOF
version: 2
charset: UTF-8
EOF

for i in {0..1023} ; do
	printf -v u "u%04d" "$i"
	cat <<-EOF

	dn: cn=$u,cn=users,dc=bridoulou,dc=fr
	changetype: add
	objectclass: inetOrgPerson
	objectclass: organizationalPerson
	objectclass: person
	objectclass: top
	cn: $u
	sn: $u
	mail: $u@bridoulou.fr
	EOF
done

for i in {0..255} ; do
	printf -v base "big%03d_" "$i"
	for g in r w d ; do
		g="${base}$g"
		cat <<-EOF

		dn: cn=$g,cn=users,dc=bridoulou,dc=fr
		changetype: add
		objectClass: groupOfNames
		objectClass: top
		cn: $g
		EOF

		for u in {0..1023} ; do
			break
			if [ $((RANDOM % 128)) -gt 0 ] ; then
				continue
			fi
			printf -v u "u%04d" "$u"
			cat <<-EOF
			member: cn=$u,cn=users,dc=bridoulou,dc=fr
			EOF
		done

		# If no user has been added, add a random one.
		if [ -n "${u#u*}" ] ; then
			printf -v u "u%04d" "$(( RANDOM % 1024 ))"
			cat <<-EOF
			member: cn=$u,cn=users,dc=bridoulou,dc=fr
			EOF
		fi
	done
done
