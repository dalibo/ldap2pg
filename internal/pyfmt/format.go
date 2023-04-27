package pyfmt

import (
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
)

type Format struct {
	Input string
	// List of either literal or field, in order.
	Sections []interface{}
	Fields   []Field
}

type Field struct {
	FieldName  string
	FormatSpec string
	Conversion string
}

func Parse(f string) (format Format, err error) {
	err = format.Parse(f)
	return
}

func (f *Format) Parse(s string) (err error) {
	f.Input = s
	end := len(s)
	start := 0
	inField := false
	for i := 0; i < end; { // Loops sections in s.
		start = i // Track the start of the section. i will move to the end.
		if inField {
			loc := strings.IndexByte(s[i:], '}')
			if -1 == loc {
				err = fmt.Errorf("end of string before end of field")
				i = end // End loop at end of step.
			} else {
				i += loc // Move before }
				field := parseField(s[start:i])
				f.Sections = append(f.Sections, field)
				f.Fields = append(f.Fields, field)
				i++ // Move after }
				inField = false
			}
		} else {
			loc := strings.IndexByte(s[i:], '{')
			if -1 == loc {
				// toto
				//     ^
				i = end // End loop at end of step.
			} else {
				// toto{titi} OR toto{{titi
				//     ^             ^
				i += loc // Move before {
				if i < end && '{' == s[i+1] {
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
	return
}

func (f Format) Format(values map[string]string) string {
	b := strings.Builder{}

	for _, item := range f.Sections {
		literal, ok := item.(string)
		if ok {
			b.WriteString(literal)
		} else {
			f := item.(Field)
			b.WriteString(values[f.FieldName])
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
