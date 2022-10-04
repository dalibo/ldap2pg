package ldap2pg

import (
	"time"

	"github.com/jackc/pgx/v5"
)

type Role struct {
	Name        string
	Comment     string
	Parents     []string
	Super       bool
	Inherit     bool
	CreateRole  bool
	CreateDB    bool
	CanLogin    bool
	Replication bool
	ConnLimit   int
	ValidUntil  time.Time
	ByPassRLS   bool
}

func NewRoleFromRow(row pgx.CollectableRow, instanceRoleColumns []string) (role Role, err error) {
	var name string
	var variableRow interface{}
	var comment string
	var parents []string
	err = row.Scan(&name, &variableRow, &comment, &parents)
	if err != nil {
		return
	}
	record := variableRow.([]interface{})
	var colname string
	for i, value := range record {
		colname = instanceRoleColumns[i]
		switch colname {
		case "rolname":
			role.Name = value.(string)
		case "rolbypassrls":
			role.ByPassRLS = value.(bool)
		case "rolcanlogin":
			role.CanLogin = value.(bool)
		case "rolconnlimit":
			role.ConnLimit = int(value.(int32))
		case "rolcreatedb":
			role.CreateDB = value.(bool)
		case "rolcreaterole":
			role.CreateRole = value.(bool)
		case "rolreplication":
			role.Replication = value.(bool)
		case "rolsuper":
			role.Super = value.(bool)
		}
	}
	return
}

func (r *Role) String() string {
	return r.Name
}
