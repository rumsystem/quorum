//go:build js && wasm
// +build js,wasm

package logging

import (
	"fmt"
	"syscall/js"

	log "github.com/ipfs/go-log/v2"
)

const DebugLevel = 0
const InfoLevel = 1
const WarnLevel = 2
const ErrorLevel = 3
const FatalLevel = 4

var logLevel = InfoLevel

type BrowserLogger struct {
	level   int
	system  string
	console js.Value
}

func (logger *BrowserLogger) Debug(args ...interface{}) {
	logger.log(DebugLevel, "", args)
}

func (logger *BrowserLogger) Debugf(format string, args ...interface{}) {
	logger.log(DebugLevel, format, args)
}

func (logger *BrowserLogger) Info(args ...interface{}) {
	logger.log(InfoLevel, "", args)
}

func (logger *BrowserLogger) Infof(format string, args ...interface{}) {
	logger.log(InfoLevel, format, args)
}

func (logger *BrowserLogger) Warn(args ...interface{}) {
	logger.log(WarnLevel, "", args)
}

func (logger *BrowserLogger) Warnf(format string, args ...interface{}) {
	logger.log(WarnLevel, format, args)
}

func (logger *BrowserLogger) Warning(args ...interface{}) {
	logger.log(WarnLevel, "", args)
}

func (logger *BrowserLogger) Warningf(format string, args ...interface{}) {
	logger.log(WarnLevel, format, args)
}

func (logger *BrowserLogger) Error(args ...interface{}) {
	logger.log(ErrorLevel, "", args)
}

func (logger *BrowserLogger) Errorf(format string, args ...interface{}) {
	logger.log(ErrorLevel, format, args)
}

func (logger *BrowserLogger) Fatal(args ...interface{}) {
	logger.log(ErrorLevel, "", args)
}

func (logger *BrowserLogger) Fatalf(format string, args ...interface{}) {
	logger.log(ErrorLevel, format, args)
}

func (logger *BrowserLogger) Panic(args ...interface{}) {
	logger.log(ErrorLevel, "", args)
}

func (logger *BrowserLogger) Panicf(format string, args ...interface{}) {
	logger.log(ErrorLevel, format, args)
}

func (logger *BrowserLogger) log(level int, format string, args ...interface{}) {
	if logLevel > level {
		return
	}
	msg := getMessage(format, args)
	msg = fmt.Sprintf("[%s] %s", logger.system, msg)
	switch level {
	case DebugLevel:
		logger.console.Call("log", msg)
	case InfoLevel:
		logger.console.Call("log", msg)
	case WarnLevel:
		logger.console.Call("warn", msg)
	case ErrorLevel:
		logger.console.Call("error", msg)
	case FatalLevel:
		logger.console.Call("error", msg)
	}
}

func getMessage(template string, fmtArgs []interface{}) string {
	if len(fmtArgs) == 0 {
		return template
	}

	if template != "" {
		return fmt.Sprintf(template, fmtArgs...)
	}

	if len(fmtArgs) == 1 {
		if str, ok := fmtArgs[0].(string); ok {
			return str
		}
	}
	return fmt.Sprint(fmtArgs...)
}

func Logger(system string) QuorumLogger {
	console := js.Global().Get("console")
	blogger := BrowserLogger{logLevel, system, console}
	return &blogger
}

func SetLogLevel(name, level string) error {
	// by system is not supported now
	return nil
}

func SetAllLoggers(lvl int) {
	if lvl < DebugLevel {
		return
	}
	logLevel = lvl
}

func LevelFromString(level string) (int, error) {
	l, e := log.LevelFromString(level)
	return int(l), e
}
