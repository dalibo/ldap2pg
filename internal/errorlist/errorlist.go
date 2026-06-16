package errorlist

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var maxErrors = 8

type List struct {
	message string

	mu     sync.Mutex
	errors []error
}

type joinUnwrapper interface {
	error
	Unwrap() []error
}

func New(message string) *List {
	return &List{message: message}
}

func (errl *List) Error() string {
	return errl.message
}

func (errl *List) Unwrap() []error {
	return errl.errors
}

// Appendf a single error to the list
//
// Returns whether list can hold more errors.
func (errl *List) Appendf(format string, args ...any) bool {
	err := fmt.Errorf(format, args...)
	if strings.Contains(format, "%w") {
		werr := errors.Unwrap(err)
		if werr == nil {
			return errl.Len() < maxErrors
		}
	}
	return errl.Append(err)
}

// Append a single error to the list
//
// use Append to continue after an error up to a number of continuable errors.
//
// Return false when list is full.
func (errl *List) Append(err error) bool {
	if err == nil {
		return true
	}

	errl.mu.Lock()
	defer errl.mu.Unlock()

	if errs, ok := errors.AsType[joinUnwrapper](err); ok {
		errl.errors = append(errl.errors, errs.Unwrap()...)
	} else {
		errl.errors = append(errl.errors, err)
	}
	return errl.Len() < maxErrors
}

func (errl *List) Len() int {
	return len(errl.errors)
}

func (errl *List) Value() error {
	switch errl.Len() {
	case 0:
		return nil
	case 1:
		return errl.errors[0]
	default:
		return errl
	}
}
