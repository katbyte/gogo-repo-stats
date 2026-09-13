// Package clog provides the shared logrus logger used by katbyte's tools.
//
// Every tool logs the same way: to stderr (stdout is reserved for the tool's
// real output, which may be machine-readable), with a text formatter and full
// timestamps, at WARN unless the tool's own environment variable raises it.
// The variable name differs per tool (TCTEST_LOG, KOI_LOG, ...), which is why
// it is passed to SetLevelFromEnv rather than baked in here.
package clog

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// TimestampFormat is the timestamp layout every logger created here uses. It is
// exported so tests and tools that build their own formatter can match it.
const TimestampFormat = "2006-01-02 15:04:05"

// DefaultLevel is the level a logger starts at before SetLevel or
// SetLevelFromEnv runs: warnings and errors only, so a tool is quiet by default.
const DefaultLevel = logrus.WarnLevel

// FallbackLevel is the level applied when SetLevel is given a string that does
// not parse. It is deliberately the most verbose level: a typo in a log level
// should show you everything rather than silently hide the debug output you
// were trying to turn on.
const FallbackLevel = logrus.TraceLevel

// Log is the process-wide logger. Packages in this module (chttp) and the tools
// that depend on it log through this single instance so one call to
// SetLevelFromEnv controls all of it.
var Log = New()

// New returns a logger configured the way every katbyte tool logs: stderr,
// text formatter, full timestamps, DefaultLevel. Log is created with it; call
// it directly only when a second, independently levelled logger is needed.
func New() *logrus.Logger {
	return NewWithOutput(os.Stderr)
}

// NewWithOutput is New writing to w instead of stderr. It exists so tests can
// capture log output without redirecting the process's stderr.
func NewWithOutput(w io.Writer) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(w)

	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = TimestampFormat
	formatter.FullTimestamp = true
	l.SetFormatter(formatter)

	l.SetLevel(DefaultLevel)

	return l
}

// SetLevelFromEnv sets Log's level from the named environment variable, for
// example "TCTEST_LOG". An unset or empty variable leaves DefaultLevel; an
// unparsable value applies FallbackLevel and logs an error naming the variable
// so the typo is visible. Tools call this once, before any logging that matters.
func SetLevelFromEnv(envVar string) {
	applyLevel(Log, os.Getenv(envVar), envVar)
}

// SetLevel sets Log's level from a level string such as "debug". It applies the
// same rules as SetLevelFromEnv (empty → DefaultLevel, unparsable →
// FallbackLevel) so a --log-level flag and the environment variable behave
// identically.
func SetLevel(level string) {
	applyLevel(Log, level, "")
}

// applyLevel applies level to l. source names where the value came from (an env
// var) purely for the error message; empty means it was passed directly.
func applyLevel(l *logrus.Logger, level, source string) {
	if level == "" {
		l.SetLevel(DefaultLevel)
		return
	}

	ll, err := logrus.ParseLevel(level)
	if err != nil {
		l.SetLevel(FallbackLevel)
		if source != "" {
			l.Errorf("defaulting to %s: unable to parse `%s` into a valid log level: %v", FallbackLevel, source, err)
		} else {
			l.Errorf("defaulting to %s: unable to parse %q into a valid log level: %v", FallbackLevel, level, err)
		}
		return
	}

	l.SetLevel(ll)
}
