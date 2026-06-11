package errorlist

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnwrap(t *testing.T) {
	r := require.New(t)

	sErr := fmt.Errorf("single")
	errs := Unwrap(sErr)
	r.Equal(1, len(errs))
	r.Equal("single", errs[0].Error())

	wErr := fmt.Errorf("wrapped: %w", sErr)
	errs = Unwrap(wErr)
	r.Equal("wrapped: single", errs[0].Error())
	r.Equal(1, len(errs))

	jErr := errors.Join(sErr, fmt.Errorf("Who shot J.R.?"))
	errs = Unwrap(jErr)
	r.Equal(2, len(errs))
	r.Equal("single", errs[0].Error())
	r.Equal("Who shot J.R.?", errs[1].Error())

	wjErr := fmt.Errorf("wraped joined : %w", jErr)
	errs = Unwrap(wjErr)
	r.Equal(2, len(errs))
	r.Equal("single", errs[0].Error())
	r.Equal("Who shot J.R.?", errs[1].Error())
}
