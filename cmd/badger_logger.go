package cmd

import (
	"fmt"
	"log"
)

type loggingLevel int

type BadgerLogger struct {
	level loggingLevel
}

const (
	DEBUG loggingLevel = iota
	INFO
	WARNING
	ERROR
)

func convertLogLevel(slevel string) loggingLevel {
	if slevel == "DEBUG" {
		return DEBUG
	}
	if slevel == "INFO" {
		return INFO
	}
	if slevel == "WARNING" {
		return WARNING
	}
	if slevel == "ERROR" {
		return ERROR
	}
	panic(fmt.Sprintf("invalid logger level, found:%s", slevel))
}

func DefaultLogger(level loggingLevel) *BadgerLogger {
	return &BadgerLogger{level: level}
}

func (l *BadgerLogger) Errorf(f string, v ...interface{}) {
	if l.level <= ERROR {
		log.Printf("INFO: "+f, v...)
	}
}

func (l *BadgerLogger) Warningf(f string, v ...interface{}) {
	if l.level <= WARNING {
		log.Printf("WARN: "+f, v...)
	}
}

func (l *BadgerLogger) Infof(f string, v ...interface{}) {
	if l.level <= INFO {
		log.Printf("INFO: "+f, v...)

	}
}

func (l *BadgerLogger) Debugf(f string, v ...interface{}) {
	if l.level <= DEBUG {
		log.Printf("DEBUG: "+f, v...)
	}
}
