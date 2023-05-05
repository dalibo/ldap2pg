package config

import (
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/slog"
)

type RoleOptions struct {
	Super       bool `column:"rolsuper" token:"SUPERUSER"`
	CreateDB    bool `column:"rolcreatedb" token:"CREATEDB"`
	CreateRole  bool `column:"rolcreaterole" token:"CREATEROLE"`
	Inherit     bool `column:"rolinherit" token:"INHERIT"`
	CanLogin    bool `column:"rolcanlogin" token:"LOGIN"`
	Replication bool `column:"rolreplication" token:"REPLICATION"`
	ByPassRLS   bool `column:"rolbypassrls" token:"BYPASSRLS"`
	ConnLimit   int  `column:"rolconnlimit" token:"CONNECTION LIMIT"`
}

func (o RoleOptions) String() string {
	v := reflect.ValueOf(o)
	t := v.Type()
	var b strings.Builder
	for _, f := range reflect.VisibleFields(t) {
		if !IsRoleColumnEnabled(f.Tag.Get("column")) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fv := v.FieldByName(f.Name)
		switch f.Type.Kind() {
		case reflect.Bool:
			o.WriteBoolOption(&b, fv.Bool(), f.Tag.Get("token"))
		case reflect.Int:
			fmt.Fprintf(&b, "%s %d", f.Tag.Get("token"), fv.Int())
		}
	}
	return b.String()
}

func (o *RoleOptions) WriteBoolOption(b *strings.Builder, value bool, token string) {
	if !value {
		b.WriteString("NO")
	}
	b.WriteString(token)
}

func (o *RoleOptions) LoadYaml(yaml map[string]interface{}) {
	for option, value := range yaml {
		switch option {
		case "SUPERUSER":
			o.Super = value.(bool)
		case "INHERIT":
			o.Inherit = value.(bool)
		case "CREATEROLE":
			o.CreateRole = value.(bool)
		case "CREATEDB":
			o.CreateDB = value.(bool)
		case "LOGIN":
			o.CanLogin = value.(bool)
		case "REPLICATION":
			o.Replication = value.(bool)
		case "BYPASSRLS":
			o.ByPassRLS = value.(bool)
		case "CONNECTION LIMIT":
			o.ConnLimit = value.(int)
		}
	}
}

func (o *RoleOptions) LoadRow(row []interface{}) {
	for i, value := range row {
		colName := GetRoleColumnNameByOrder(i)
		switch colName {
		case "rolbypassrls":
			o.ByPassRLS = value.(bool)
		case "rolcanlogin":
			o.CanLogin = value.(bool)
		case "rolconnlimit":
			o.ConnLimit = int(value.(int32))
		case "rolcreatedb":
			o.CreateDB = value.(bool)
		case "rolcreaterole":
			o.CreateRole = value.(bool)
		case "rolinherit":
			o.Inherit = value.(bool)
		case "rolreplication":
			o.Replication = value.(bool)
		case "rolsuper":
			o.Super = value.(bool)
		}
	}
}

// Global state of role columns in inspected instance.
var instanceRoleColumns struct {
	availability map[string]bool
	order        []string
}

func ProcessRoleColumns(columns []string, super bool) {
	instanceRoleColumns.order = columns
	instanceRoleColumns.availability = make(map[string]bool)
	t := reflect.TypeOf(RoleOptions{})
	for _, f := range reflect.VisibleFields(t) {
		instanceRoleColumns.availability[f.Tag.Get("column")] = false
	}
	for _, name := range columns {
		if !super && ("rolsuper" == name || "rolreplication" == name || "rolbypassrls" == name) {
			slog.Debug("Ignoring privileged role column", "column", name)
			continue
		}
		instanceRoleColumns.availability[name] = true
	}
}

func GetRoleColumnsOrder() []string {
	return instanceRoleColumns.order
}

func GetRoleColumnNameByOrder(order int) string {
	return instanceRoleColumns.order[order]
}

func IsRoleColumnEnabled(name string) bool {
	available, ok := instanceRoleColumns.availability[name]
	return ok && available
}
