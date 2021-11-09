//go:build js && wasm
// +build js,wasm

package wasm

import (
	"syscall/js"
)

func Promisefy(fn func() (map[string]interface{}, error)) js.Value {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]
		go func() {
			ret, err := fn()
			if err != nil {
				reject.Invoke(err.Error())
			}
			resolve.Invoke(js.ValueOf(ret))
		}()
		// The handler of a Promise doesn't return any value
		return nil
	})
	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}
