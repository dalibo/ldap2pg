package errorlist

import "errors"

// Unwrap ensures an error list
//
// Either return a single item list with err or
// unwrap compound errors.
func Unwrap(err error) []error {
	if err == nil {
		return nil
	}

	var errs []error
	if errl, ok := errors.AsType[interface {
		error
		Unwrap() []error
	}](err); ok {
		errs = errl.Unwrap()
	} else {
		errs = []error{err}
	}
	return errs
}
