package errorlist

var maxErrors = 8

type List struct {
	errors  []error
	message string
}

type joinedErrors interface {
	Unwrap() []error
}

func New(message string) *List {
	return &List{message: message}
}

func (list List) Error() string {
	return list.message
}

func (list List) Unwrap() []error {
	return list.errors
}

// Append a single error to the list
//
// use Append to continue after an error up to a number of continuable errors.
//
// Return false when list is full.
// Panics if error wraps multiple errors.
// Use Extend() to append joined errors.
func (list *List) Append(err error) bool {
	if _, ok := err.(joinedErrors); ok {
		panic("errorlist: cannot append agreggated error")
	}
	if err != nil {
		list.errors = append(list.errors, err)
	}
	return list.Len() < maxErrors
}

// Extend list with wrapped errors.
//
// Use Extend to aggregate skippable joined errors or fail fast on single error.
//
// Return nil if list has free slots.
// Return err as is if it's a single error.
// Return self if list is full.
func (list *List) Extend(err error) error {
	if errs, ok := err.(joinedErrors); ok {
		list.errors = append(list.errors, errs.Unwrap()...)
	} else {
		return err
	}

	if list.Len() >= maxErrors {
		return list
	}
	return nil
}

func (list List) Len() int {
	return len(list.errors)
}
