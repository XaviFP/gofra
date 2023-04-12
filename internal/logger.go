package gofra

import (
	"io"
	"log"
	"os"
)

type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

type Logger struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	err   *log.Logger

	logLevel LogLevel
}

func NewLogger(debug bool) Logger {
	flags := log.Ldate | log.Ltime | log.Llongfile
	logger := Logger{
		debug:    log.New(io.Discard, "DEBUG ", flags),
		info:     log.New(os.Stdout, "INFO ", flags),
		warn:     log.New(os.Stderr, "WARN ", flags),
		err:      log.New(os.Stderr, "ERROR ", flags),
		logLevel: LogLevelInfo,
	}

	if debug {
		logger.logLevel = LogLevelDebug
		logger.debug.SetOutput(os.Stderr)
	}

	return logger
}

func (l Logger) Info(message string) {
	if l.logLevel >= LogLevelInfo {
		l.info.Println(message)
	}
}

func (l Logger) Debug(message string) {
	if l.logLevel >= LogLevelDebug {
		l.debug.Println(message)
	}
}

func (l Logger) Warn(message string) {
	if l.logLevel >= LogLevelWarn {
		l.warn.Println(message)
	}
}

func (l Logger) Error(message string) {
	if l.logLevel >= LogLevelError {
		l.err.Println(message)
	}
}

type xmlWriter struct {
	logger *log.Logger
}

func (xw xmlWriter) Write(p []byte) (int, error) {
	xw.logger.Printf("%s", p)

	return len(p), nil
}

func getStreamLoggers(logXML bool) (io.Writer, io.Writer) {
	var xmlIn, xmlOut io.Writer
	if logXML {
		xmlIn = xmlWriter{log.New(os.Stdout, "IN ", log.LstdFlags)}
		xmlOut = xmlWriter{log.New(os.Stdout, "OUT ", log.LstdFlags)}
	}

	return xmlIn, xmlOut
}
