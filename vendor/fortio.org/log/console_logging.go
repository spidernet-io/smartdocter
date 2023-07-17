// Copyright 2023 Fortio Authors
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

package log

import (
	"os"
	"time"
)

// to avoid making a new package/namespace for colors, we use a struct.
type color struct {
	Reset     string
	Red       string
	Green     string
	Yellow    string
	Blue      string
	Purple    string
	Cyan      string
	Gray      string
	White     string
	BrightRed string
	DarkGray  string
}

var (
	// these should really be constants but go doesn't have constant structs, arrays etc...

	// ANSI color codes.
	Colors = color{
		Reset:     "\033[0m",
		Red:       "\033[31m",
		Green:     "\033[32m",
		Yellow:    "\033[33m",
		Blue:      "\033[34m",
		Purple:    "\033[35m",
		Cyan:      "\033[36m",
		Gray:      "\033[37m",
		White:     "\033[97m",
		BrightRed: "\033[91m",
		DarkGray:  "\033[90m",
	}

	// Mapping of log levels to color.
	LevelToColor = []string{
		Colors.Gray,
		Colors.Cyan,
		Colors.Green,
		Colors.Yellow,
		Colors.Red,
		Colors.Purple,
		Colors.BrightRed,
	}
	// Cached flag for whether to use color output or not.
	Color = false
)

// ConsoleLogging is a utility to check if the current logger output is a console (terminal).
func ConsoleLogging() bool {
	f, ok := jsonWriter.(*os.File)
	if !ok {
		return false
	}
	s, _ := f.Stat()
	return (s.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// SetColorMode computes whether we currently should be using color text mode or not.
// Need to be reset if config changes (but is already automatically re evaluated when calling SetOutput()).
func SetColorMode() {
	Color = ColorMode()
}

// ColorMode returns true if we should be using color text mode, which is either because it's
// forced or because we are in a console and the config allows it.
// Should not be called often, instead read/update the Color variable when needed.
func ColorMode() bool {
	return Config.ForceColor || (Config.ConsoleColor && ConsoleLogging())
}

func colorTimestamp() string {
	if Config.NoTimestamp {
		return ""
	}
	return time.Now().Format(Colors.DarkGray + "15:04:05.000 ")
}
