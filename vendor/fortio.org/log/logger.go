// Copyright 2017-2023 Fortio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Fortio's log is simple logger built on top of go's default one with
additional opinionated levels similar to glog but simpler to use and configure.

See [Config] object for options like whether to include line number and file name of caller or not etc
*/
package log // import "fortio.org/log"

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Level is the level of logging (0 Debug -> 6 Fatal).
type Level int32

// Log levels. Go can't have variable and function of the same name so we keep
// medium length (Dbg,Info,Warn,Err,Crit,Fatal) names for the functions.
const (
	Debug Level = iota
	Verbose
	Info
	Warning
	Error
	Critical
	Fatal
)

//nolint:revive // we keep "Config" for the variable itself.
type LogConfig struct {
	LogPrefix      string    // "Prefix to log lines before logged messages
	LogFileAndLine bool      // Logs filename and line number of callers to log.
	FatalPanics    bool      // If true, log.Fatalf will panic (stack trace) instead of just exit 1
	FatalExit      func(int) // Function to call upon log.Fatalf. e.g. os.Exit.
	JSON           bool      // If true, log in structured JSON format instead of text.
	NoTimestamp    bool      // If true, don't log timestamp in json.
}

// DefaultConfig() returns the default initial configuration for the logger, best suited
// for servers. It will log caller file and line number, use a prefix to split line info
// from the message and panic (+exit) on Fatal.
func DefaultConfig() *LogConfig {
	return &LogConfig{
		LogPrefix:      "> ",
		LogFileAndLine: true,
		FatalPanics:    true,
		FatalExit:      os.Exit,
		JSON:           true,
	}
}

var (
	Config = DefaultConfig()
	// Used for dynamic flag setting as strings and validation.
	LevelToStrA = []string{
		"Debug",
		"Verbose",
		"Info",
		"Warning",
		"Error",
		"Critical",
		"Fatal",
	}
	levelToStrM   map[string]Level
	levelInternal int32
	// Used for JSON logging.
	LevelToJSON = []string{
		// matching https://github.com/grafana/grafana/blob/main/docs/sources/explore/logs-integration.md
		// adding the "" around to save processing when generating json. using short names to save some bytes.
		"\"dbug\"",
		"\"trace\"",
		"\"info\"",
		"\"warn\"",
		"\"err\"",
		"\"crit\"",
		"\"fatal\"",
	}
)

// SetDefaultsForClientTools changes the default value of LogPrefix and LogFileAndLine
// to make output without caller and prefix, a default more suitable for command line tools (like dnsping).
// Needs to be called before flag.Parse(). Caller could also use log.Printf instead of changing this
// if not wanting to use levels. Also makes log.Fatalf just exit instead of panic.
func SetDefaultsForClientTools() {
	Config.LogPrefix = ""
	Config.LogFileAndLine = false
	Config.FatalPanics = false
	Config.JSON = false
}

// JSONEntry is the logical format of the JSON [Config.JSON] output mode.
// While that serialization of is custom in order to be cheap, it maps to the following
// structure.
type JSONEntry struct {
	TS    int64 // in microseconds since epoch (unix micros)
	Level string
	File  string
	Line  int
	Msg   string
	// + additional optional fields
	// See https://go.dev/play/p/oPK5vyUH2tf for a possibility (using https://github.com/devnw/ajson )
	// or https://go.dev/play/p/H0RPmuc3dzv (using github.com/mitchellh/mapstructure)
}

// LogEntry Ts to time.Time conversion.
// The returned time is set UTC to avoid TZ mismatch.
func (l *JSONEntry) Time() time.Time {
	const million = int64(1e6)
	return time.Unix(
		l.TS/million,        // Microseconds -> Seconds
		1000*(l.TS%million), // Microseconds -> Nanoseconds
	)
}

//nolint:gochecknoinits // needed
func init() {
	setLevel(Info) // starting value
	levelToStrM = make(map[string]Level, 2*len(LevelToStrA))
	for l, name := range LevelToStrA {
		// Allow both -loglevel Verbose and -loglevel verbose ...
		levelToStrM[name] = Level(l)
		levelToStrM[strings.ToLower(name)] = Level(l)
	}
	log.SetFlags(log.Ltime)
}

func setLevel(lvl Level) {
	atomic.StoreInt32(&levelInternal, int32(lvl))
}

// String returns the string representation of the level.
func (l Level) String() string {
	return LevelToStrA[l]
}

// ValidateLevel returns error if the level string is not valid.
func ValidateLevel(str string) (Level, error) {
	var lvl Level
	var ok bool
	if lvl, ok = levelToStrM[str]; !ok {
		return -1, fmt.Errorf("should be one of %v", LevelToStrA)
	}
	return lvl, nil
}

// LoggerStaticFlagSetup call to setup a static flag under the passed name or
// `-loglevel` by default, to set the log level.
// Use https://pkg.go.dev/fortio.org/dflag/dynloglevel#LoggerFlagSetup for a dynamic flag instead.
func LoggerStaticFlagSetup(names ...string) {
	if len(names) == 0 {
		names = []string{"loglevel"}
	}
	for _, name := range names {
		flag.Var(&flagV, name, fmt.Sprintf("log `level`, one of %v", LevelToStrA))
	}
}

// --- Start of code/types needed string to level custom flag validation section ---

type flagValidation struct {
	ours bool
}

var flagV = flagValidation{true}

func (f *flagValidation) String() string {
	// Need to tell if it's our value or the zeroValue the flag package creates
	// to decide whether to print (default ...) or not.
	if !f.ours {
		return ""
	}
	return GetLogLevel().String()
}

func (f *flagValidation) Set(inp string) error {
	v := strings.ToLower(strings.TrimSpace(inp))
	lvl, err := ValidateLevel(v)
	if err != nil {
		return err
	}
	SetLogLevel(lvl)
	return nil
}

// --- End of code/types needed string to level custom flag validation section ---

// Sets level from string (called by dflags).
// Use https://pkg.go.dev/fortio.org/dflag/dynloglevel#LoggerFlagSetup to set up
// `-loglevel` as a dynamic flag (or an example of how this function is used).
func SetLogLevelStr(str string) error {
	var lvl Level
	var err error
	if lvl, err = ValidateLevel(str); err != nil {
		return err
	}
	SetLogLevel(lvl)
	return err // nil
}

// SetLogLevel sets the log level and returns the previous one.
func SetLogLevel(lvl Level) Level {
	return setLogLevel(lvl, true)
}

// SetLogLevelQuiet sets the log level and returns the previous one but does
// not log the change of level itself.
func SetLogLevelQuiet(lvl Level) Level {
	return setLogLevel(lvl, false)
}

// setLogLevel sets the log level and returns the previous one.
// if logChange is true the level change is logged.
func setLogLevel(lvl Level, logChange bool) Level {
	prev := GetLogLevel()
	if lvl < Debug {
		log.Printf("SetLogLevel called with level %d lower than Debug!", lvl)
		return -1
	}
	if lvl > Critical {
		log.Printf("SetLogLevel called with level %d higher than Critical!", lvl)
		return -1
	}
	if lvl != prev {
		if logChange {
			logPrintf(Info, "Log level is now %d %s (was %d %s)", lvl, lvl.String(), prev, prev.String())
		}
		setLevel(lvl)
	}
	return prev
}

// GetLogLevel returns the currently configured LogLevel.
func GetLogLevel() Level {
	return Level(atomic.LoadInt32(&levelInternal))
}

// Log returns true if a given level is currently logged.
func Log(lvl Level) bool {
	return int32(lvl) >= atomic.LoadInt32(&levelInternal)
}

// LevelByName returns the LogLevel by its name.
func LevelByName(str string) Level {
	return levelToStrM[str]
}

// Logf logs with format at the given level.
// 2 level of calls so it's always same depth for extracting caller file/line.
// Note that log.Logf(Fatal, "...") will not panic or exit, only log.Fatalf() does.
func Logf(lvl Level, format string, rest ...interface{}) {
	logPrintf(lvl, format, rest...)
}

// Used when doing our own logging writing, in JSON/structured mode.
var (
	jsonWriter      io.Writer = os.Stderr
	jsonWriterMutex sync.Mutex
)

func jsonWrite(msg string) {
	jsonWriterMutex.Lock()
	_, _ = jsonWriter.Write([]byte(msg)) // if we get errors while logging... can't quite ... log errors
	jsonWriterMutex.Unlock()
}

func TimeToTS(t time.Time) int64 {
	return t.UnixMicro()
}

func jsonTimestamp() string {
	if Config.NoTimestamp {
		return ""
	}
	return fmt.Sprintf("\"ts\":%d,", TimeToTS(time.Now()))
}

func logPrintf(lvl Level, format string, rest ...interface{}) {
	if !Log(lvl) {
		return
	}
	if Config.LogFileAndLine { //nolint:nestif
		_, file, line, _ := runtime.Caller(2)
		file = file[strings.LastIndex(file, "/")+1:]
		if Config.JSON {
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,\"file\":%q,\"line\":%d,\"msg\":%q}\n",
				jsonTimestamp(), LevelToJSON[lvl], file, line, fmt.Sprintf(format, rest...)))
		} else {
			log.Print(LevelToStrA[lvl][0:1], " ", file, ":", line, Config.LogPrefix, fmt.Sprintf(format, rest...))
		}
	} else {
		if Config.JSON {
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,\"msg\":%q}\n",
				jsonTimestamp(), LevelToJSON[lvl], fmt.Sprintf(format, rest...)))
		} else {
			log.Print(LevelToStrA[lvl][0:1], " ", Config.LogPrefix, fmt.Sprintf(format, rest...))
		}
	}
}

// Printf forwards to the underlying go logger to print (with only timestamp prefixing).
func Printf(format string, rest ...interface{}) {
	log.Printf(format, rest...)
}

// SetOutput sets the output to a different writer (forwards to system logger).
func SetOutput(w io.Writer) {
	jsonWriter = w
	log.SetOutput(w)
}

// SetFlags forwards flags to the system logger.
func SetFlags(f int) {
	log.SetFlags(f)
}

// -- would be nice to be able to create those in a loop instead of copypasta:

// Debugf logs if Debug level is on.
func Debugf(format string, rest ...interface{}) {
	logPrintf(Debug, format, rest...)
}

// LogVf logs if Verbose level is on.
func LogVf(format string, rest ...interface{}) { //nolint:revive
	logPrintf(Verbose, format, rest...)
}

// Infof logs if Info level is on.
func Infof(format string, rest ...interface{}) {
	logPrintf(Info, format, rest...)
}

// Warnf logs if Warning level is on.
func Warnf(format string, rest ...interface{}) {
	logPrintf(Warning, format, rest...)
}

// Errf logs if Warning level is on.
func Errf(format string, rest ...interface{}) {
	logPrintf(Error, format, rest...)
}

// Critf logs if Warning level is on.
func Critf(format string, rest ...interface{}) {
	logPrintf(Critical, format, rest...)
}

// Fatalf logs if Warning level is on and panics or exits.
func Fatalf(format string, rest ...interface{}) {
	logPrintf(Fatal, format, rest...)
	if Config.FatalPanics {
		panic("aborting...")
	}
	Config.FatalExit(1)
}

// FErrF logs a fatal error and returns 1.
// meant for cli main functions written like:
//
//	func main() { os.Exit(Main()) }
//
// and in Main() they can do:
//
//	if err != nil {
//		return log.FErrf("error: %v", err)
//	}
//
// so they can be tested with testscript.
// See https://github.com/fortio/delta/ for an example.
func FErrf(format string, rest ...interface{}) int {
	logPrintf(Fatal, format, rest...)
	return 1
}

// LogDebug shortcut for fortio.Log(fortio.Debug).
func LogDebug() bool { //nolint:revive
	return Log(Debug)
}

// LogVerbose shortcut for fortio.Log(fortio.Verbose).
func LogVerbose() bool { //nolint:revive
	return Log(Verbose)
}

// LoggerI defines a log.Logger like interface to pass to packages
// for simple logging. See [Logger()].
type LoggerI interface {
	Printf(format string, rest ...interface{})
}

type loggerShm struct{}

func (l *loggerShm) Printf(format string, rest ...interface{}) {
	logPrintf(Info, format, rest...)
}

// Logger returns a LoggerI (standard logger compatible) that can be used for simple logging.
func Logger() LoggerI {
	logger := loggerShm{}
	return &logger
}

// Somewhat slog compatible/style logger

type KeyVal struct {
	Key   string
	Value fmt.Stringer
}

type StringValue string

func (s StringValue) String() string {
	return string(s)
}

func Str(key, value string) KeyVal {
	return KeyVal{Key: key, Value: StringValue(value)}
}

type ValueTypes interface{ any }

type ValueType[T ValueTypes] struct {
	Val T
}

func (v *ValueType[T]) String() string {
	return fmt.Sprint(v.Val)
}

func Attr[T ValueTypes](key string, value T) KeyVal {
	return KeyVal{
		Key:   key,
		Value: &ValueType[T]{Val: value},
	}
}

func S(lvl Level, msg string, attrs ...KeyVal) {
	if !Log(lvl) {
		return
	}
	buf := strings.Builder{}
	var format string
	if Config.JSON {
		format = ",%q:%q"
	} else {
		format = ", %s=%s"
	}
	for _, attr := range attrs {
		buf.WriteString(fmt.Sprintf(format, attr.Key, attr.Value.String()))
	}
	if Config.LogFileAndLine { //nolint:nestif
		_, file, line, _ := runtime.Caller(1)
		file = file[strings.LastIndex(file, "/")+1:]
		if Config.JSON {
			// TODO share code with log.Printf yet without extra locks or allocations/buffers?
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,\"file\":%q,\"line\":%d,\"msg\":%q%s}\n",
				jsonTimestamp(), LevelToJSON[lvl], file, line, msg, buf.String()))
		} else {
			log.Print(LevelToStrA[lvl][0:1], " ", file, ":", line, Config.LogPrefix, msg, buf.String())
		}
	} else {
		if Config.JSON {
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,\"msg\":%q%s}\n",
				jsonTimestamp(), LevelToJSON[lvl], msg, buf.String()))
		} else {
			log.Print(LevelToStrA[lvl][0:1], " ", Config.LogPrefix, msg, buf.String())
		}
	}
}
