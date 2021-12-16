// +build js,wasm
// from https://github.com/hack-pad/go-indexeddb/blob/main/idb/internal/exception/exception.go

package exception

import (
	"fmt"
	"syscall/js"
)

// Catch recovers from panics and attempts to convert the value into an error.
// Must be used directly in a defer statement, can not be called elsewhere.
// Set 'err' to the address of the return value, typically with a named return error value.
// Example: defer exception.Catch(&err)
func Catch(err *error) {
	recoverErr := handleRecovery(recover())
	if recoverErr != nil {
		*err = recoverErr
	}
}

// CatchHandler is the same as Catch, but enables custom error handling after recovering.
func CatchHandler(fn func(err error)) {
	err := handleRecovery(recover())
	if err != nil {
		fn(err)
	}
}

func handleRecovery(r interface{}) error {
	if r == nil {
		return nil
	}
	switch val := r.(type) {
	case error:
		return val
	case js.Value:
		return js.Error{Value: val}
	default:
		return fmt.Errorf("%+v", val)
	}
}
