package errorlist_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dalibo/ldap2pg/internal/errorlist"
	"github.com/stretchr/testify/require"
)

var numError = 0

func TestErrorListExtend(t *testing.T) {
	r := require.New(t)
	list := errorlist.New("test extend")

	r.Nil(list.Extend(nil))

	errs := errors.Join(buildErrors(2)...)
	r.Nil(list.Extend(errs))
	r.Equal(2, list.Len())

	errs2 := errors.Join(buildErrors(6)...)
	r.NotNil(list.Extend(errs2))
	r.Equal(8, list.Len())

	unaggregateError := errors.New("unaggregate error")
	r.NotNil(list.Extend(unaggregateError))
}

func TestErrorListAppend(t *testing.T) {
	r := require.New(t)
	list := errorlist.New("test append")

	r.True(list.Append(nil))

	errs := errors.Join(buildErrors(7)...)
	r.Panics(func() { list.Append(errs) })
	for _, err := range errs.(interface{ Unwrap() []error }).Unwrap() {
		r.True(list.Append(err))
	}
	r.Equal(7, list.Len())

	r.False(list.Append(errors.New("error 8")))
	r.Equal(8, list.Len())
}

func buildErrors(n int) []error {
	var errors []error
	for i := 1; i <= n; i++ {
		err := fmt.Errorf("error %d", numError+i)
		errors = append(errors, err)
	}
	return errors
}
