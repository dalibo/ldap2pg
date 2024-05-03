#!/bin/bash

set -eu

cat <<-EOF
version: 6

postgres:
  databases_query: [big0, big1, big2, big3]
  managed_roles_query: |
    SELECT 'public'
    UNION
    SELECT DISTINCT role.rolname
    FROM pg_roles AS role
    LEFT OUTER JOIN pg_auth_members AS ms ON ms.member = role.oid
    LEFT OUTER JOIN pg_roles AS ldap_roles
      ON ldap_roles.rolname = 'ldap_roles' AND ldap_roles.oid = ms.roleid
    WHERE role.rolname = 'ldap_roles'
        OR ldap_roles.oid IS NOT NULL
    ORDER BY 1;


privileges:
  read:
  - __connect__
  - __usage_on_schemas__
  - __select_on_tables__
  - __select_on_sequences__

  write:
  - __temporary__
  - __execute_on_functions__
  - __insert_on_tables__
  - __delete_on_tables__
  - __update_on_tables__
  - __update_on_sequences__
  - __usage_on_sequences__
  - __trigger_on_tables__
  - __truncate_on_tables__
  - __references_on_tables__

  define:
  - __create_on_schemas__

rules:
- description: "Base roles"
  roles:
  - name: ldap_roles
    comment: All roles managed by ldap2pg
EOF

for n in {0..255} ; do
	printf -v n "%03d" "$n"
	cat <<-EOF

	- description: "Define groups and privileges for schema $n."
	  roles:
	  - name: big${n}_r
	    parents: ldap_roles
	  - name: big${n}_w
	    parents:
	    - ldap_roles
	    - big${n}_r
	  - name: big${n}_d
	    parents:
	    - ldap_roles
	    - big${n}_w
	  grants:
	  - privilege: read
	    role: big${n}_r
	    schemas: nsp$n
	  - privilege: write
	    role: big${n}_w
	    schemas: nsp$n
	  - privilege: define
	    role: big${n}_d
	    schemas: nsp$n
	EOF
done

cat <<-EOF

- description: "Define roles from directory."
  ldapsearch:
    base: cn=users,dc=bridoulou,dc=fr
    filter: (cn=big*)
  roles:
    name: "{member.cn}"
    parents:
    - ldap_roles
    - "{cn}"
EOF
