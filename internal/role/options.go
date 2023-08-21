package role

import (
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/slog"
)

type Options struct {
	Super       bool `column:"rolsuper" token:"SUPERUSER"`
	CreateDB    bool `column:"rolcreatedb" token:"CREATEDB"`
	CreateRole  bool `column:"rolcreaterole" token:"CREATEROLE"`
	Inherit     bool `column:"rolinherit" token:"INHERIT"`
	CanLogin    bool `column:"rolcanlogin" token:"LOGIN"`
	Replication bool `column:"rolreplication" token:"REPLICATION"`
	ByPassRLS   bool `column:"rolbypassrls" token:"BYPASSRLS"`
	ConnLimit   int  `column:"rolconnlimit" token:"CONNECTION LIMIT"`
}

func (o Options) String() string {
	v := reflect.ValueOf(o)
	t := v.Type()
	var b strings.Builder
	for _, f := range reflect.VisibleFields(t) {
		if !isColumnEnabled(f.Tag.Get("column")) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fv := v.FieldByName(f.Name)
		switch f.Type.Kind() {
		case reflect.Bool:
			writeBoolOption(&b, fv.Bool(), f.Tag.Get("token"))
		case reflect.Int:
			fmt.Fprintf(&b, "%s %d", f.Tag.Get("token"), fv.Int())
		}
	}
	return b.String()
}

func (o Options) Diff(other Options) string {
	v := reflect.ValueOf(o)
	otherV := reflect.ValueOf(other)
	t := v.Type()
	var b strings.Builder
	for _, f := range reflect.VisibleFields(t) {
		if !isColumnEnabled(f.Tag.Get("column")) {
			continue
		}
		fv := v.FieldByName(f.Name)
		otherFV := otherV.FieldByName(f.Name)
		switch f.Type.Kind() {
		case reflect.Bool:
			if fv.Bool() != otherFV.Bool() {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				writeBoolOption(&b, fv.Bool(), f.Tag.Get("token"))
			}
		case reflect.Int:
			i := fv.Int()
			if i != otherFV.Int() {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%s %d", f.Tag.Get("token"), i)
			}
		}
	}
	return b.String()
}

func (o *Options) LoadYaml(yaml map[string]interface{}) {
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

func (o *Options) LoadRow(row []interface{}) {
	for i, value := range row {
		colName := getColumnNameByOrder(i)
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
var instanceColumns struct {
	availability map[string]bool
	order        []string
}

func ProcessColumns(columns []string, super bool) {
	instanceColumns.order = columns
	instanceColumns.availability = make(map[string]bool)
	t := reflect.TypeOf(Options{})
	for _, f := range reflect.VisibleFields(t) {
		instanceColumns.availability[f.Tag.Get("column")] = false
	}
	for _, name := range columns {
		if !super && ("rolsuper" == name || "rolreplication" == name || "rolbypassrls" == name) {
			slog.Debug("Ignoring privileged role column", "column", name)
			continue
		}
		instanceColumns.availability[name] = true
	}
}

func getColumnNameByOrder(order int) string {
	return instanceColumns.order[order]
}

func isColumnEnabled(name string) bool {
	available, ok := instanceColumns.availability[name]
	return ok && available
}

func writeBoolOption(b *strings.Builder, value bool, token string) {
	if !value {
		b.WriteString("NO")
	}
	b.WriteString(token)
}
