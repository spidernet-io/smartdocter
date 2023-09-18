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
	"strings"
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

// LogRequest logs the incoming request, TLSInfo,
// including headers when loglevel is verbose.
// additional key:value pairs can be passed as extraAttributes.
//
//nolint:revive
func LogRequest(r *http.Request, msg string, extraAttributes ...KeyVal) {
	if !Log(Info) {
		return
	}
	attr := []KeyVal{
		Str("method", r.Method), Attr("url", r.URL), Str("proto", r.Proto),
		Str("remote_addr", r.RemoteAddr), Str("host", r.Host),
		Str("header.x-forwarded-proto", r.Header.Get("X-Forwarded-Proto")),
		Str("header.x-forwarded-for", r.Header.Get("X-Forwarded-For")),
		Str("user-agent", r.Header.Get("User-Agent")),
	}
	attr = AppendTLSInfoAttrs(attr, r)
	attr = append(attr, extraAttributes...)
	if LogVerbose() {
		// Host is removed from headers map and put separately
		for name, headers := range r.Header {
			attr = append(attr, Str("header."+name, strings.Join(headers, ",")))
		}
	}
	S(Info, msg, attr...)
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
