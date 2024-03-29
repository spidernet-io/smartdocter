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

// A small utility to just serve the hostname on TCP and/or UDP.

package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// CmdServeHostname is used by agnhost Cobra.
var CmdServeHostname = &cobra.Command{
	Use:   "serve-hostname",
	Short: "Serves the hostname",
	Long:  `Serves the hostname through HTTP / TCP / UDP on the given port.`,
	// Args:  cobra.MaximumNArgs(0),
	Run: rootmain,
}

var (
	doTCP   bool
	doUDP   bool
	doHTTP  bool
	doClose bool
	port    int
)

func init() {
	CmdServeHostname.Flags().BoolVar(&doTCP, "tcp", false, "Serve raw over TCP.")
	CmdServeHostname.Flags().BoolVar(&doUDP, "udp", false, "Serve raw over UDP.")
	CmdServeHostname.Flags().BoolVar(&doHTTP, "http", true, "Serve HTTP.")
	CmdServeHostname.Flags().BoolVar(&doClose, "close", false, "Close connection per each HTTP request.")
	CmdServeHostname.Flags().IntVar(&port, "port", 80, "Port number.")
}

func rootmain(cmd *cobra.Command, args []string) {
	if doHTTP && (doTCP || doUDP) {
		log.Fatalf("Can't server TCP/UDP mode and HTTP mode at the same time")
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error from os.Hostname(): %s", err)
	}

	if doTCP {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("Error from net.Listen(): %s", err)
		}
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Fatalf("Error from Accept(): %s", err)
				}
				log.Printf("TCP request from %s", conn.RemoteAddr().String())
				res := "server: " + hostname + ", client: " + conn.RemoteAddr().String()
				_, e := conn.Write([]byte(res))
				if e != nil {
					log.Fatalf("Error from Write(): %s", e)
				}
				conn.Close()
			}
		}()
	}
	if doUDP {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("Error from net.ResolveUDPAddr(): %s", err)
		}
		sock, err := net.ListenUDP("udp", addr)
		if err != nil {
			log.Fatalf("Error from ListenUDP(): %s", err)
		}
		go func() {
			var buffer [16]byte
			for {
				_, cliAddr, err := sock.ReadFrom(buffer[0:])
				if err != nil {
					log.Fatalf("Error from ReadFrom(): %s", err)
				}
				res := "server: " + hostname + ", client: " + cliAddr.String()
				log.Printf("UDP request from %s", cliAddr.String())
				_, e := sock.WriteTo([]byte(res), cliAddr)
				if e != nil {
					log.Fatalf("Error from Write(): %s", e)
				}
			}
		}()
	}
	if doHTTP {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP request from %s", r.RemoteAddr)
			log.Printf("HTTP request: %+v \n", r)

			if doClose {
				// Add this header to force to close the connection after serving the request.
				w.Header().Add("Connection", "close")
			}
			message := "{ \n"
			message += fmt.Sprintf("  \"server\": \"%s\"\n", hostname)
			message += fmt.Sprintf("  \"clientIp\": \"%s\"\n", r.RemoteAddr)
			message += "}"
			fmt.Fprintf(w, "%s", message)

			message = "\n"
			message += "{ \n"
			message += fmt.Sprintf("  \"request method\": \"%s\"\n", r.Method)
			message += fmt.Sprintf("  \"request url\": \"%s\"\n", r.RequestURI)
			message += "  \"request header\": {\n"
			for k, v := range r.Header {
				message += fmt.Sprintf("      \"%s\": \"%s\"\n", k, v)
			}
			message += "  }\n"
			message += "}"
			fmt.Fprintf(w, "%s", message)

		})
		go func() {
			// Run in a closure so http.ListenAndServe doesn't block
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		}()
	}
	log.Printf("Serving on port %d.\n", port)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	sig := <-signals
	log.Printf("Shutting down after receiving signal: %s.\n", sig)
	log.Printf("Awaiting pod deletion.\n")
	time.Sleep(60 * time.Second)
}

func main() {
	if err := CmdServeHostname.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
