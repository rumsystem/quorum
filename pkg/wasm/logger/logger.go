//go:build js && wasm
// +build js,wasm

package logger

import "syscall/js"

type Logger struct {
	console js.Value
}

var debug = false

var Console = NewLogger()

func SetDebug(enable bool) {
	debug = enable
}

func NewLogger() *Logger {
	console := js.Global().Get("console")
	return &Logger{console}
}

func (logger *Logger) Log(msg string) {
	logger.console.Call("log", msg)
}

func (logger *Logger) Debug(msg string) {
	if debug {
		logger.console.Call("log", msg)
	}
}

func (logger *Logger) Warn(msg string) {
	logger.console.Call("warn", msg)
}

func (logger *Logger) Error(msg string) {
	logger.console.Call("error", msg)
}
