//go:build js && wasm
// +build js,wasm

package wasm

import "syscall/js"

type Logger struct {
	console js.Value
}

func NewLogger() *Logger {
	console := js.Global().Get("console")
	return &Logger{console}
}

func (logger *Logger) Log(msg string) {
	logger.console.Call("log", msg)
}

func (logger *Logger) Warn(msg string) {
	logger.console.Call("warn", msg)
}

func (logger *Logger) Error(msg string) {
	logger.console.Call("error", msg)
}
