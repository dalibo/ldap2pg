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
