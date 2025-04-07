package role

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

type Options struct {
	Super       bool `column:"rolsuper" mapstructure:"SUPERUSER"`
	CreateDB    bool `column:"rolcreatedb" mapstructure:"CREATEDB"`
	CreateRole  bool `column:"rolcreaterole" mapstructure:"CREATEROLE"`
	Inherit     bool `column:"rolinherit" mapstructure:"INHERIT"`
	CanLogin    bool `column:"rolcanlogin" mapstructure:"LOGIN"`
	Replication bool `column:"rolreplication" mapstructure:"REPLICATION"`
	ByPassRLS   bool `column:"rolbypassrls" mapstructure:"BYPASSRLS"`
	ConnLimit   int  `column:"rolconnlimit" mapstructure:"CONNECTION LIMIT"`
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
			writeBoolOption(&b, fv.Bool(), f.Tag.Get("mapstructure"))
		case reflect.Int:
			fmt.Fprintf(&b, "%s %d", f.Tag.Get("mapstructure"), fv.Int())
		}
	}
	return b.String()
}

// Diff returns the SQL to match wanted role options
func (o Options) Diff(wanted Options) string {
	v := reflect.ValueOf(o)
	wantedV := reflect.ValueOf(wanted)
	var b strings.Builder
	for _, f := range reflect.VisibleFields(v.Type()) {
		if !isColumnEnabled(f.Tag.Get("column")) {
			continue
		}
		fv := v.FieldByName(f.Name)
		wantedFV := wantedV.FieldByName(f.Name)
		switch f.Type.Kind() {
		case reflect.Bool:
			if fv.Bool() != wantedFV.Bool() {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				writeBoolOption(&b, wantedFV.Bool(), f.Tag.Get("mapstructure"))
			}
		case reflect.Int:
			i := wantedFV.Int()
			if i != fv.Int() {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%s %d", f.Tag.Get("mapstructure"), i)
			}
		}
	}
	return b.String()
}

func (o *Options) LoadRow(row []any) {
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

func ProcessColumns(columns []string, super bool) []string {
	instanceColumns.availability = make(map[string]bool)
	t := reflect.TypeOf(Options{})
	for _, f := range reflect.VisibleFields(t) {
		instanceColumns.availability[f.Tag.Get("column")] = false
	}
	for _, name := range columns {
		if !super && (name == "rolsuper" || name == "rolreplication" || name == "rolbypassrls") {
			slog.Debug("Ignoring privileged role column", "column", name)
			continue
		}
		instanceColumns.availability[name] = true
		instanceColumns.order = append(instanceColumns.order, name)
	}
	return instanceColumns.order
}

func getColumnNameByOrder(order int) string {
	return instanceColumns.order[order]
}

func isColumnEnabled(name string) bool {
	available, ok := instanceColumns.availability[name]
	return ok && available
}

func writeBoolOption(b *strings.Builder, value bool, keyword string) {
	if !value {
		b.WriteString("NO")
	}
	b.WriteString(keyword)
}
