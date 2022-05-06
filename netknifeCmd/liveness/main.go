// Copyright Authors of Kubernetes.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2014 The Kubernetes Authors.

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

// A simple server that is alive for 10 seconds, then reports unhealthy for
// the rest of its (hopefully) short existence.

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// CmdLiveness is used by agnhost Cobra.
var CmdLiveness = &cobra.Command{
	Use:   "liveness",
	Short: "Starts a server that is alive for 10 seconds",
	Long:  "A simple server that is alive for 10 seconds, then reports unhealthy for the rest of its (hopefully) short existence",
	Args:  cobra.MaximumNArgs(0),
	Run:   rootmain,
}

func rootmain(cmd *cobra.Command, args []string) {
	started := time.Now()
	http.HandleFunc("/started", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data := (time.Since(started)).String()
		_, e := w.Write([]byte(data))
		if e != nil {
			log.Fatalf("Error from Write(): %s", e)
		}
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		duration := time.Since(started)
		if duration.Seconds() > 10 {
			w.WriteHeader(500)
			_, e := w.Write([]byte(fmt.Sprintf("error: %v", duration.Seconds())))
			if e != nil {
				log.Fatalf("Error from Write(): %s", e)
			}
		} else {
			w.WriteHeader(200)
			_, e := w.Write([]byte("ok"))
			if e != nil {
				log.Fatalf("Error from Write(): %s", e)
			}
		}
	})
	http.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		loc, err := url.QueryUnescape(r.URL.Query().Get("loc"))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid redirect: %q", r.URL.Query().Get("loc")), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, loc, http.StatusFound)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	if err := CmdLiveness.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
