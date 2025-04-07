package cmd

import "os"

// Custom error handling exit code.
//
// os.Exit() bypasses deferred functions.
// This error allows passing exit code as an error
// to execute deferred functions before exiting in caller.
type errorCode struct {
	code    int
	message string
}

func (err errorCode) Error() string {
	return err.message
}

func (err errorCode) Exit() {
	os.Exit(err.code)
}
