package pyfmt

import (
	"errors"
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
)

type Format struct {
	Input string
	// List of either literal or field, in order.
	Sections []any
	Fields   []*Field
}

func (f Format) IsStatic() bool {
	return len(f.Fields) == 0
}

type Field struct {
	FieldName  string
	FormatSpec string
	Conversion string
	Method     string
}

func Parse(f string) (format Format, err error) {
	err = format.Parse(f)
	return
}

func (f *Format) Parse(s string) (err error) {
	f.Input = s
	var (
		end     = len(s)
		inField = false
		next    byte
		start   int
	)

	for i := 0; i < end; { // Loops sections in s.
		start = i // Track the start of the section. i will move to the end.
		if inField {
			loc := strings.IndexByte(s[i:], '}')
			if loc == -1 {
				err = fmt.Errorf("end of string before end of field")
				i = end // End loop at end of step.
			} else {
				i += loc // Move before }
				field := parseField(s[start:i])
				f.Sections = append(f.Sections, &field)
				f.Fields = append(f.Fields, &field)
				i++ // Move after }
				inField = false
			}
		} else {
			loc := strings.IndexByte(s[i:], '{')
			if loc == -1 {
				// toto
				//     ^
				i = end // End loop at end of step.
			} else {
				// toto{titi} OR toto{{titi
				//     ^             ^
				i += loc // Move before {
				if i < end-1 {
					next = s[i+1]
				} else {
					next = 0
				}
				if i < end && next == '{' {
					// To escape {{, send two strings, the one before the second { and the rest after on next iteration.
					//
					// toto{{titi
					//      ^
					i++ // Move before second { to include first { as literal { this step.
				} else {
					inField = true
				}
			}
			if i > start { // Avoid empty literal.
				f.Sections = append(f.Sections, strings.ReplaceAll(s[start:i], "}}", "}"))
			}
			// toto{titi OR toto{{titi
			//      ^             ^
			i++ // Move after {, literal or escape.
		}
	}
	if inField {
		err = errors.New("unexpected end of format")
	}
	return
}

func parseField(s string) (f Field) {
	before, after, found := strings.Cut(s, "!")
	if found {
		// case {0!r} OR {0!r:>30}
		f.FieldName = before
		before, after, _ = strings.Cut(after, ":")
		f.Conversion = before
	} else {
		// case {0} OR {0:>30}
		before, after, _ = strings.Cut(before, ":")
		f.FieldName = before
	}
	f.FormatSpec = after
	if strings.HasSuffix(f.FieldName, "()") {
		lastPoint := strings.LastIndex(f.FieldName, ".")
		f.Method = f.FieldName[lastPoint+1:]
		f.FieldName = f.FieldName[:lastPoint]
	}
	return
}

func (f Format) Format(values map[string]string) string {
	if values == nil {
		if !f.IsStatic() {
			panic("rendering dynamic format without values")
		}
		return f.String()
	}

	b := strings.Builder{}

	for _, item := range f.Sections {
		literal, ok := item.(string)
		if ok {
			b.WriteString(literal)
		} else {
			f := item.(*Field)
			v := values[f.FieldName]
			switch f.Method {
			case "":
			case "lower()":
				v = strings.ToLower(v)
			case "upper()":
				v = strings.ToUpper(v)
			case "identifier()":
				v = fmt.Sprintf("\"%s\"", v)
			case "string()":
				v = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
			default:
				v = "!INVALID_METHOD"
			}
			b.WriteString(v)
		}
	}
	return b.String()
}

func (f Format) String() string {
	return f.Input
}

func ListExpressions(fmts ...Format) []string {
	set := mapset.NewSet[string]()
	for _, f := range fmts {
		for _, field := range f.Fields {
			set.Add(field.FieldName)
		}
	}
	return set.ToSlice()
}

// Extract root variables references by expressions.
func ListVariables(expressions ...string) []string {
	attrSet := mapset.NewSet[string]()
	for _, expr := range expressions {
		attr, _, _ := strings.Cut(expr, ".")
		attrSet.Add(attr)
	}
	return attrSet.ToSlice()
}
