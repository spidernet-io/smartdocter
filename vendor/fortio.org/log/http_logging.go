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

package log

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

// TLSInfo returns ' https <cipher suite> "<peer CN>"' if the request is using TLS
// (and ' "<peer CN>"' part if mtls / a peer certificate is present) or "" otherwise.
// Use [AppendTLSInfoAttrs] unless you do want to just output text.
func TLSInfo(r *http.Request) string {
	if r.TLS == nil {
		return ""
	}
	cliCert := ""
	if len(r.TLS.PeerCertificates) > 0 {
		cliCert = fmt.Sprintf(" %q", r.TLS.PeerCertificates[0].Subject)
	}
	return fmt.Sprintf(" https %s%s", tls.CipherSuiteName(r.TLS.CipherSuite), cliCert)
}

func AppendTLSInfoAttrs(attrs []KeyVal, r *http.Request) []KeyVal {
	if r.TLS == nil {
		return attrs
	}
	attrs = append(attrs, Attr("tls", true))
	if len(r.TLS.PeerCertificates) > 0 {
		attrs = append(attrs, Str("tls.peer_cn", r.TLS.PeerCertificates[0].Subject.CommonName))
	}
	return attrs
}

func AddIfNotEmpty(attrs []KeyVal, key, value string) []KeyVal {
	if value != "" {
		attrs = append(attrs, Str(key, value))
	}
	return attrs
}

// LogRequest logs the incoming request, TLSInfo,
// including headers when loglevel is verbose.
// additional key:value pairs can be passed as extraAttributes.
//
//nolint:revive // name is fine.
func LogRequest(r *http.Request, msg string, extraAttributes ...KeyVal) {
	if !Log(Info) {
		return
	}
	// URL struct is quite verbose and not that interesting to log all pieces so we log the String() version
	var url KeyVal
	if r.URL == nil {
		url = Any("url", r.URL) // basically 'null'
	} else {
		url = Str("url", r.URL.String())
	}
	attr := []KeyVal{
		Str("method", r.Method), url, Str("host", r.Host),
		Str("proto", r.Proto), Str("remote_addr", r.RemoteAddr),
	}
	if !LogVerbose() { // in verbose all headers are already logged
		attr = AddIfNotEmpty(attr, "user-agent", r.Header.Get("User-Agent"))
		// note this only prints the first one, while verbose mode will join all values with ','
		attr = AddIfNotEmpty(attr, "header.x-forwarded-proto", r.Header.Get("X-Forwarded-Proto"))
		attr = AddIfNotEmpty(attr, "header.x-forwarded-for", r.Header.Get("X-Forwarded-For"))
		attr = AddIfNotEmpty(attr, "header.x-forwarded-host", r.Header.Get("X-Forwarded-Host"))
	}
	attr = AppendTLSInfoAttrs(attr, r)
	attr = append(attr, extraAttributes...)
	if LogVerbose() {
		// Host is removed from headers map and put separately
		// Need to sort to get a consistent order
		keys := make([]string, 0, len(r.Header))
		for name := range r.Header {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			nl := strings.ToLower(name)
			if nl != "user-agent" {
				nl = "header." + nl
			}
			attr = append(attr, Str(nl, strings.Join(r.Header[name], ",")))
		}
	}
	// not point in having the line number be this file
	s(Info, false, Config.JSON, msg, attr...)
}

// LogResponse logs the response code, byte size and duration of the request.
// additional key:value pairs can be passed as extraAttributes.
//
//nolint:revive // name is fine.
func LogResponse[T *ResponseRecorder | *http.Response](r T, msg string, extraAttributes ...KeyVal) {
	if !Log(Info) {
		return
	}
	var status int
	var size int64
	switch v := any(r).(type) { // go generics...
	case *ResponseRecorder:
		status = v.StatusCode
		size = v.ContentLength
	case *http.Response:
		status = v.StatusCode
		size = v.ContentLength
	}
	attr := []KeyVal{
		Int("status", status),
		Int64("size", size),
	}
	attr = append(attr, extraAttributes...)
	// not point in having the line number be this file
	s(Info, false, Config.JSON, msg, attr...)
}

// Can be used (and is used by LogAndCall()) to wrap a http.ResponseWriter to record status code and size.
type ResponseRecorder struct {
	w             http.ResponseWriter
	startTime     time.Time
	StatusCode    int
	ContentLength int64
}

func (rr *ResponseRecorder) Header() http.Header {
	return rr.w.Header()
}

func (rr *ResponseRecorder) Write(p []byte) (int, error) {
	size, err := rr.w.Write(p)
	rr.ContentLength += int64(size)
	if err != nil {
		rr.StatusCode = http.StatusInternalServerError
	} else if rr.StatusCode == 0 {
		rr.StatusCode = http.StatusOK
	}
	return size, err
}

func (rr *ResponseRecorder) WriteHeader(code int) {
	rr.w.WriteHeader(code)
	rr.StatusCode = code
}

// Implement http.Flusher interface.
func (rr *ResponseRecorder) Flush() {
	if f, ok := rr.w.(http.Flusher); ok {
		f.Flush()
	}
}

// LogAndCall logs the incoming request and the response code, byte size and duration
// of the request.
//
// If Config.CombineRequestAndResponse or the LOGGER_COMBINE_REQUEST_AND_RESPONSE
// environment variable is true, then a single log entry is done combining request and
// response information, including catching for panic.
//
// Additional key:value pairs can be passed as extraAttributes.
//
//nolint:revive // name is fine.
func LogAndCall(msg string, handlerFunc http.HandlerFunc, extraAttributes ...KeyVal) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is really 2 functions but we want to be able to change config without rewiring the middleware
		if Config.CombineRequestAndResponse {
			respRec := &ResponseRecorder{w: w, startTime: time.Now()}
			defer func() {
				if err := recover(); err != nil {
					s(Critical, false, Config.JSON, "panic in handler", Any("error", err))
					if Log(Verbose) {
						s(Verbose, false, Config.JSON, "stack trace", Str("stack", string(debug.Stack())))
					}
				}
				attr := []KeyVal{
					Int("status", respRec.StatusCode),
					Int64("size", respRec.ContentLength),
					Int64("microsec", time.Since(respRec.startTime).Microseconds()),
				}
				attr = append(attr, extraAttributes...)
				LogRequest(r, msg, attr...)
			}()
			handlerFunc(respRec, r)
			return
		}
		LogRequest(r, msg, extraAttributes...)
		respRec := &ResponseRecorder{w: w, startTime: time.Now()}
		handlerFunc(respRec, r)
		LogResponse(respRec, msg, Int64("microsec", time.Since(respRec.startTime).Microseconds()))
	})
}

type logWriter struct {
	source string
	level  Level
}

// Returns a Std logger that will log to the given level with the given source attribute.
// Can be passed for instance to net/http/httputil.ReverseProxy.ErrorLog.
func NewStdLogger(source string, level Level) *log.Logger {
	return log.New(logWriter{source, level}, "", 0)
}

func (w logWriter) Write(p []byte) (n int, err error) {
	// Force JSON to avoid infinite loop and also skip file/line so it doesn't show this file as the source
	// (TODO consider passing the level up the stack to look for the caller)
	s(w.level, false, true, strings.TrimSpace(string(p)), Str("src", w.source))
	return len(p), nil
}

// InterceptStandardLogger changes the output of the standard logger to use ours, at the given
// level, with the source "std", as a catchall.
func InterceptStandardLogger(level Level) {
	log.SetFlags(0)
	log.SetOutput(logWriter{"std", level})
}
