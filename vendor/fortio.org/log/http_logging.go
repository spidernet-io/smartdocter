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
	"net/http"
)

// TLSInfo returns " https <cipher suite>" if the request is using TLS, or "" otherwise.
func TLSInfo(r *http.Request) string {
	if r.TLS == nil {
		return ""
	}
	return fmt.Sprintf(" https %s", tls.CipherSuiteName(r.TLS.CipherSuite))
}

// LogRequest logs the incoming request, including headers when loglevel is verbose.
//
//nolint:revive
func LogRequest(r *http.Request, msg string) {
	if Log(Info) {
		tlsInfo := TLSInfo(r)
		Printf("%s: %v %v %v %v (%v) %s %q%s", msg, r.Method, r.URL, r.Proto, r.RemoteAddr,
			r.Header.Get("X-Forwarded-Proto"), r.Header.Get("X-Forwarded-For"), r.Header.Get("User-Agent"), tlsInfo)
	}
	if LogVerbose() {
		// Host is removed from headers map and put separately
		Printf("Header Host: %v", r.Host)
		for name, headers := range r.Header {
			for _, h := range headers {
				Printf("Header %v: %v\n", name, h)
			}
		}
	}
}
