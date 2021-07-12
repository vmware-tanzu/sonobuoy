/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errlog

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	// DebugOutput controls whether to output the trace of every error
	DebugOutput = false

	// loglevel used for sirupsen/logrus
	LogLevel logLevelFlagType = "info"
)

type logLevelFlagType string

func (l *logLevelFlagType) String() string { return string(*l) }
func (l *logLevelFlagType) Type() string   { return "level" }
func (l *logLevelFlagType) Set(str string) error {
	*l = logLevelFlagType(str)
	return SetLevel(str)
}

func SetLevel(s string) error {
	// Just using debug to set log level for as long
	// as we want to keep the deprecated flag.
	if DebugOutput {
		LogLevel = "debug"
	}
	switch s {
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
		DebugOutput = true
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
		DebugOutput = true
	default:
		return fmt.Errorf("unknown log level %q", s)
	}

	return nil

}

// LogError logs an error, optionally with a tracelog
func LogError(err error) {
	if DebugOutput {
		// Print the error message with the stack trace (%+v) in the "trace" field
		logrus.WithField("trace", fmt.Sprintf("%+v", err)).Error(err)
	} else {
		logrus.Error(err.Error())
	}
}
