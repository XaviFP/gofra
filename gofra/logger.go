package gofra

import (
	"io"
	"log"
	"os"
)

type Logger struct {
	Info  *log.Logger
	Debug *log.Logger
	Warn  *log.Logger
	Error *log.Logger
}

func NewLogger(debug bool) Logger {
	flags := log.Ldate | log.Ltime | log.Llongfile
	logger := Logger{
		Info:  log.New(os.Stdout, "INFO ", flags),
		Debug: log.New(io.Discard, "DEBUG ", flags),
		Warn:  log.New(os.Stderr, "WARN ", flags),
		Error: log.New(os.Stderr, "ERROR ", flags),
	}

	if debug {
		logger.Debug.SetOutput(os.Stderr)
	}

	return logger

}

type xmlWriter struct {
	logger *log.Logger
}

func (xw xmlWriter) Write(p []byte) (int, error) {
	// TODO: this Write method should write to gofra's logger
	// and make use of Info log level
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
