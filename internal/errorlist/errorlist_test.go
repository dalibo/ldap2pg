package errorlist_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/errorlist"
	"github.com/stretchr/testify/require"
)

func TestErrorListAppend(t *testing.T) {
	r := require.New(t)
	list := errorlist.New("test append")

	r.True(list.Append(nil))
	r.Equal(0, list.Len())

	r.True(list.Appendf("wrap: %w", nil))
	r.Equal(0, list.Len())

	errs := errors.Join(buildErrors(7)...)
	r.True(list.Append(errs))
	r.Equal(7, list.Len())

	r.False(list.Append(errors.New("error 8")))
	r.Equal(8, list.Len())
}

func buildErrors(n int) []error {
	var errors []error
	for i := 1; i <= n; i++ {
		err := fmt.Errorf("error %d", i)
		errors = append(errors, err)
	}
	return errors
}

func TestErrorListValue(t *testing.T) {
	r := require.New(t)

	err := errorlist.New("context")
	err.Append(fmt.Errorf("content"))
	errs := errorlist.Unwrap(err.Value())
	r.Equal(1, len(errs))
	r.Equal("context: content", errs[0].Error())

	err.Append(fmt.Errorf("content 2"))
	errs = errorlist.Unwrap(err.Value())
	r.Equal(3, len(errs))
	r.Equal("context", errs[0].Error())
	r.Equal("content", errs[1].Error())
	r.Equal("content 2", errs[2].Error())
}
