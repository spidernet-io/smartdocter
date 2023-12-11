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

So far it's a "global" logger as in you just use the functions in the package directly (e.g log.S())
and the configuration is global for the process.
*/
package log // import "fortio.org/log"

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fortio.org/log/goroutine"
	"fortio.org/struct2env"
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
	NoLevel
	// Prefix for all config from environment,
	// e.g NoTimestamp becomes LOGGER_NO_TIMESTAMP.
	EnvPrefix = "LOGGER_"
)

//nolint:revive // we keep "Config" for the variable itself.
type LogConfig struct {
	LogPrefix      string    // "Prefix to log lines before logged messages
	LogFileAndLine bool      // Logs filename and line number of callers to log.
	FatalPanics    bool      // If true, log.Fatalf will panic (stack trace) instead of just exit 1
	FatalExit      func(int) `env:"-"` // Function to call upon log.Fatalf. e.g. os.Exit.
	JSON           bool      // If true, log in structured JSON format instead of text (but see ConsoleColor).
	NoTimestamp    bool      // If true, don't log timestamp in json.
	ConsoleColor   bool      // If true and we detect console output (not redirected), use text+color mode.
	// Force color mode even if logger output is not console (useful for CI that recognize ansi colors).
	// SetColorMode() must be called if this or ConsoleColor are changed.
	ForceColor bool
	// If true, log the goroutine ID (gid) in json.
	GoroutineID bool
	// If true, single combined log for LogAndCall
	CombineRequestAndResponse bool
	// String version of the log level, used for setting from environment.
	Level string
}

// DefaultConfig() returns the default initial configuration for the logger, best suited
// for servers. It will log caller file and line number, use a prefix to split line info
// from the message and panic (+exit) on Fatal.
// It's JSON structured by default, unless console is detected.
// Use SetDefaultsForClientTools for CLIs.
func DefaultConfig() *LogConfig {
	return &LogConfig{
		LogPrefix:                 "> ",
		LogFileAndLine:            true,
		FatalPanics:               true,
		FatalExit:                 os.Exit,
		JSON:                      true,
		ConsoleColor:              true,
		GoroutineID:               true,
		CombineRequestAndResponse: true,
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
		"\"info\"", // For Printf / NoLevel JSON output
	}
	// Reverse mapping of level string used in JSON to Level. Used by https://github.com/fortio/logc
	// to interpret and colorize pre existing JSON logs.
	JSONStringLevelToLevel map[string]Level
)

// SetDefaultsForClientTools changes the default value of LogPrefix and LogFileAndLine
// to make output without caller and prefix, a default more suitable for command line tools (like dnsping).
// Needs to be called before flag.Parse(). Caller could also use log.Printf instead of changing this
// if not wanting to use levels. Also makes log.Fatalf just exit instead of panic.
func SetDefaultsForClientTools() {
	Config.LogPrefix = " "
	Config.LogFileAndLine = false
	Config.FatalPanics = false
	Config.ConsoleColor = true
	Config.JSON = false
	Config.GoroutineID = false
	Config.CombineRequestAndResponse = false
	SetColorMode()
}

// JSONEntry is the logical format of the JSON [Config.JSON] output mode.
// While that serialization of is custom in order to be cheap, it maps to the following
// structure.
type JSONEntry struct {
	TS    float64 // In seconds since epoch (unix micros resolution), see TimeToTS().
	R     int64   // Goroutine ID (if enabled)
	Level string
	File  string
	Line  int
	Msg   string
	// + additional optional fields
	// See https://go.dev/play/p/oPK5vyUH2tf for a possibility (using https://github.com/devnw/ajson )
	// or https://go.dev/play/p/H0RPmuc3dzv (using github.com/mitchellh/mapstructure)
}

// Time() converts a LogEntry.TS to time.Time.
// The returned time is set UTC to avoid TZ mismatch.
// Inverse of TimeToTS().
func (l *JSONEntry) Time() time.Time {
	sec := int64(l.TS)
	return time.Unix(
		sec, // float seconds -> int Seconds
		int64(math.Round(1e6*(l.TS-float64(sec)))*1000), // reminder -> Nanoseconds
	)
}

//nolint:gochecknoinits // needed
func init() {
	setLevel(Info) // starting value
	levelToStrM = make(map[string]Level, 2*len(LevelToStrA))
	JSONStringLevelToLevel = make(map[string]Level, len(LevelToJSON)-1) // -1 to not reverse info to NoLevel
	for l, name := range LevelToStrA {
		// Allow both -loglevel Verbose and -loglevel verbose ...
		levelToStrM[name] = Level(l)
		levelToStrM[strings.ToLower(name)] = Level(l)
	}
	for l, name := range LevelToJSON[0 : Fatal+1] { // Skip NoLevel
		// strip the quotes around
		JSONStringLevelToLevel[name[1:len(name)-1]] = Level(l)
	}
	log.SetFlags(log.Ltime)
	configFromEnv()
	SetColorMode()
	jWriter.buf.Grow(2048)
}

func configFromEnv() {
	prev := Config.Level
	struct2env.SetFromEnv(EnvPrefix, Config)
	if Config.Level != "" && Config.Level != prev {
		lvl, err := ValidateLevel(Config.Level)
		if err != nil {
			Errf("Invalid log level from environment %q: %v", Config.Level, err)
			return
		}
		SetLogLevelQuiet(lvl)
		Infof("Log level set from environment %s%s to %s", EnvPrefix, "LEVEL", lvl.String())
	}
	Config.Level = GetLogLevel().String()
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
		logUnconditionalf(Config.LogFileAndLine, Error, "SetLogLevel called with level %d lower than Debug!", lvl)
		return -1
	}
	if lvl > Critical {
		logUnconditionalf(Config.LogFileAndLine, Error, "SetLogLevel called with level %d higher than Critical!", lvl)
		return -1
	}
	if lvl != prev {
		if logChange && Log(Info) {
			logUnconditionalf(Config.LogFileAndLine, Info, "Log level is now %d %s (was %d %s)", lvl, lvl.String(), prev, prev.String())
		}
		setLevel(lvl)
		Config.Level = lvl.String()
	}
	return prev
}

// EnvHelp shows the current config as environment variables.
//
// LOGGER_LOG_PREFIX, LOGGER_LOG_FILE_AND_LINE, LOGGER_FATAL_PANICS,
// LOGGER_JSON, LOGGER_NO_TIMESTAMP, LOGGER_CONSOLE_COLOR, LOGGER_CONSOLE_COLOR
// LOGGER_FORCE_COLOR, LOGGER_GOROUTINE_ID, LOGGER_COMBINE_REQUEST_AND_RESPONSE,
// LOGGER_LEVEL.
func EnvHelp(w io.Writer) {
	res, _ := struct2env.StructToEnvVars(Config)
	str := struct2env.ToShellWithPrefix(EnvPrefix, res, true)
	fmt.Fprintln(w, "# Logger environment variables:")
	fmt.Fprint(w, str)
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
	jWriter = jsonWriter{w: os.Stderr, tsBuf: make([]byte, 0, 32)}
)

type jsonWriter struct {
	w     io.Writer
	mutex sync.Mutex
	buf   bytes.Buffer
	tsBuf []byte
}

func jsonWrite(msg string) {
	jsonWriteBytes([]byte(msg))
}

func jsonWriteBytes(msg []byte) {
	jWriter.mutex.Lock()
	_, _ = jWriter.w.Write(msg) // if we get errors while logging... can't quite ... log errors
	jWriter.mutex.Unlock()
}

// Converts a time.Time to a float64 timestamp (seconds since epoch at microsecond resolution).
// This is what is used in JSONEntry.TS.
func TimeToTS(t time.Time) float64 {
	// note that nanos like 1688763601.199999400 become 1688763601.1999996 in float64 (!)
	// so we use UnixMicro to hide this problem which also means we don't give the nearest
	// microseconds but it gets truncated instead ( https://go.dev/play/p/rzojmE2odlg )
	usec := t.UnixMicro()
	tfloat := float64(usec) / 1e6
	return tfloat
}

// timeToTStr is copying the string-ification code from jsonTimestamp(),
// it is used by tests to individually test what jsonTimestamp does.
func timeToTStr(t time.Time) string {
	return fmt.Sprintf("%.6f", TimeToTS(t))
}

func jsonTimestamp() string {
	if Config.NoTimestamp {
		return ""
	}
	// Change timeToTStr if changing this.
	return fmt.Sprintf("\"ts\":%.6f,", TimeToTS(time.Now()))
}

// Returns the json GoRoutineID if enabled.
func jsonGID() string {
	if !Config.GoroutineID {
		return ""
	}
	return fmt.Sprintf("\"r\":%d,", goroutine.ID())
}

func logPrintf(lvl Level, format string, rest ...interface{}) {
	if !Log(lvl) {
		return
	}
	if Config.JSON && !Config.LogFileAndLine && !Color && !Config.NoTimestamp && !Config.GoroutineID && len(rest) == 0 {
		logSimpleJSON(lvl, format)
		return
	}
	logUnconditionalf(Config.LogFileAndLine, lvl, format, rest...)
}

func logSimpleJSON(lvl Level, msg string) {
	jWriter.mutex.Lock()
	jWriter.buf.Reset()
	jWriter.buf.WriteString("{\"ts\":")
	t := TimeToTS(time.Now())
	jWriter.tsBuf = jWriter.tsBuf[:0] // reset the slice
	jWriter.tsBuf = strconv.AppendFloat(jWriter.tsBuf, t, 'f', 6, 64)
	jWriter.buf.Write(jWriter.tsBuf)
	fmt.Fprintf(&jWriter.buf, ",\"level\":%s,\"msg\":%q}\n",
		LevelToJSON[lvl],
		msg)
	_, _ = jWriter.w.Write(jWriter.buf.Bytes())
	jWriter.mutex.Unlock()
}

func logUnconditionalf(logFileAndLine bool, lvl Level, format string, rest ...interface{}) {
	prefix := Config.LogPrefix
	if prefix == "" {
		prefix = " "
	}
	lvl1Char := ""
	if lvl == NoLevel {
		prefix = ""
	}
	if logFileAndLine { //nolint:nestif
		_, file, line, _ := runtime.Caller(3)
		file = file[strings.LastIndex(file, "/")+1:]
		if Color {
			jsonWrite(fmt.Sprintf("%s%s%s %s:%d%s%s%s%s\n",
				colorTimestamp(), colorGID(), ColorLevelToStr(lvl),
				file, line, prefix, LevelToColor[lvl], fmt.Sprintf(format, rest...), Colors.Reset))
		} else if Config.JSON {
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,%s\"file\":%q,\"line\":%d,\"msg\":%q}\n",
				jsonTimestamp(), LevelToJSON[lvl], jsonGID(), file, line, fmt.Sprintf(format, rest...)))
		} else {
			if lvl != NoLevel {
				lvl1Char = "[" + LevelToStrA[lvl][0:1] + "]"
			}
			log.Print(lvl1Char, " ", file, ":", line, prefix, fmt.Sprintf(format, rest...))
		}
	} else {
		if Color {
			jsonWrite(fmt.Sprintf("%s%s%s%s%s%s%s\n",
				colorTimestamp(), colorGID(), ColorLevelToStr(lvl), prefix, LevelToColor[lvl],
				fmt.Sprintf(format, rest...), Colors.Reset))
		} else if Config.JSON {
			if len(rest) != 0 {
				format = fmt.Sprintf(format, rest...)
			}
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,%s\"msg\":%q}\n",
				jsonTimestamp(), LevelToJSON[lvl], jsonGID(), format))
		} else {
			if lvl != NoLevel {
				lvl1Char = "[" + LevelToStrA[lvl][0:1] + "]"
			}
			log.Print(lvl1Char, prefix, fmt.Sprintf(format, rest...))
		}
	}
}

// Printf forwards to the underlying go logger to print (with only timestamp prefixing).
func Printf(format string, rest ...interface{}) {
	logUnconditionalf(false, NoLevel, format, rest...)
}

// SetOutput sets the output to a different writer (forwards to system logger).
func SetOutput(w io.Writer) {
	jWriter.w = w
	log.SetOutput(w)
	SetColorMode() // Colors.Reset color mode boolean
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
// for simple logging. See [Logger()]. See also [NewStdLogger()] for
// intercepting with same type / when an interface can't be used.
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
	Key      string
	StrValue string
	Value    fmt.Stringer
	Cached   bool
}

// String() is the slog compatible name for Str. Ends up calling Any() anyway.
func String(key, value string) KeyVal {
	return Any(key, value)
}

func Str(key, value string) KeyVal {
	return Any(key, value)
}

// Few more slog style short cuts.
func Int(key string, value int) KeyVal {
	return Any(key, value)
}

func Int64(key string, value int64) KeyVal {
	return Any(key, value)
}

func Float64(key string, value float64) KeyVal {
	return Any(key, value)
}

func Bool(key string, value bool) KeyVal {
	return Any(key, value)
}

func (v *KeyVal) StringValue() string {
	if !v.Cached {
		v.StrValue = v.Value.String()
		v.Cached = true
	}
	return v.StrValue
}

type ValueTypes interface{ any }

type ValueType[T ValueTypes] struct {
	Val T
}

func toJSON(v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return strconv.Quote(fmt.Sprintf("ERR marshaling %v: %v", v, err))
	}
	str := string(bytes)
	// This is kinda hacky way to handle both structured and custom serialization errors, and
	// struct with no public fields for which we need to call Error() to get a useful string.
	if e, isError := v.(error); isError && str == "{}" {
		return fmt.Sprintf("%q", e.Error())
	}
	return str
}

func (v ValueType[T]) String() string {
	// if the type is numeric, use Sprint(v.val) otherwise use Sprintf("%q", v.Val) to quote it.
	switch s := any(v.Val).(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprint(s)
	case string:
		return fmt.Sprintf("%q", s)
	/* It's all handled by json fallback now even though slightly more expensive at runtime, it's a lot simpler */
	default:
		return toJSON(v.Val) // was fmt.Sprintf("%q", fmt.Sprint(v.Val))
	}
}

// Our original name, now switched to slog style Any.
func Attr[T ValueTypes](key string, value T) KeyVal {
	return Any(key, value)
}

func Any[T ValueTypes](key string, value T) KeyVal {
	return KeyVal{
		Key:   key,
		Value: ValueType[T]{Val: value},
	}
}

// S logs a message of the given level with additional attributes.
func S(lvl Level, msg string, attrs ...KeyVal) {
	s(lvl, Config.LogFileAndLine, Config.JSON, msg, attrs...)
}

func s(lvl Level, logFileAndLine bool, json bool, msg string, attrs ...KeyVal) {
	if !Log(lvl) {
		return
	}
	if Config.JSON && !Config.LogFileAndLine && !Color && !Config.NoTimestamp && !Config.GoroutineID && len(attrs) == 0 {
		logSimpleJSON(lvl, msg)
		return
	}
	buf := strings.Builder{}
	var format string
	if Color {
		format = Colors.Reset + ", " + Colors.Blue + "%s" + Colors.Reset + "=" + LevelToColor[lvl] + "%v"
	} else if json {
		format = ",%q:%s"
	} else {
		format = ", %s=%s"
	}
	for _, attr := range attrs {
		buf.WriteString(fmt.Sprintf(format, attr.Key, attr.StringValue()))
	}
	// TODO share code with log.logUnconditionalf yet without extra locks or allocations/buffers?
	prefix := Config.LogPrefix
	if prefix == "" {
		prefix = " "
	}
	lvl1Char := ""
	if lvl == NoLevel {
		prefix = ""
	} else {
		lvl1Char = "[" + LevelToStrA[lvl][0:1] + "]"
	}
	if logFileAndLine { //nolint:nestif
		_, file, line, _ := runtime.Caller(2)
		file = file[strings.LastIndex(file, "/")+1:]
		if Color {
			jsonWrite(fmt.Sprintf("%s%s%s %s:%d%s%s%s%s%s\n",
				colorTimestamp(), colorGID(), ColorLevelToStr(lvl),
				file, line, prefix, LevelToColor[lvl], msg, buf.String(), Colors.Reset))
		} else if json {
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,%s\"file\":%q,\"line\":%d,\"msg\":%q%s}\n",
				jsonTimestamp(), LevelToJSON[lvl], jsonGID(), file, line, msg, buf.String()))
		} else {
			log.Print(lvl1Char, " ", file, ":", line, prefix, msg, buf.String())
		}
	} else {
		if Color {
			jsonWrite(fmt.Sprintf("%s%s%s%s%s%s%s%s\n",
				colorTimestamp(), colorGID(), ColorLevelToStr(lvl), prefix, LevelToColor[lvl], msg, buf.String(), Colors.Reset))
		} else if json {
			jsonWrite(fmt.Sprintf("{%s\"level\":%s,\"msg\":%q%s}\n",
				jsonTimestamp(), LevelToJSON[lvl], msg, buf.String()))
		} else {
			log.Print(lvl1Char, prefix, msg, buf.String())
		}
	}
}
