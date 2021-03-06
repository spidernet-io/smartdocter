// Copyright Authors of Kubernetes.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2022 The Kubernetes Authors.

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

// Package grpchealthchecking offers a tiny grpc health checking endpoint.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"net/http"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// CmdGrpcHealthChecking is used by agnhost Cobra.
var CmdGrpcHealthChecking = &cobra.Command{
	Use:   "grpc-health-checking",
	Short: "Starts a simple grpc health checking endpoint",
	Long:  "Starts a simple grpc health checking endpoint with --port to serve on and --service to check status for. The endpoint returns SERVING for the first --delay-unhealthy-sec, and NOT_SERVING after this. NOT_FOUND will be returned for the requests for non-configured service name. Probe can be forced to be set NOT_SERVING by calling /make-not-serving http endpoint.",
	Args:  cobra.MaximumNArgs(0),
	Run:   rootmain,
}

var (
	port              int
	httpPort          int
	delayUnhealthySec int
	service           string
	forceUnhealthy    *bool
)

func init() {
	CmdGrpcHealthChecking.Flags().IntVar(&port, "port", 5000, "Port number.")
	CmdGrpcHealthChecking.Flags().IntVar(&httpPort, "http-port", 8080, "Port number for the /make-serving and /make-not-serving.")
	CmdGrpcHealthChecking.Flags().IntVar(&delayUnhealthySec, "delay-unhealthy-sec", -1, "Number of seconds to delay before start reporting NOT_SERVING, negative value indicates never.")
	CmdGrpcHealthChecking.Flags().StringVar(&service, "service", "", "Service name to register the health check for.")
	forceUnhealthy = nil
}

type HealthChecker struct {
	started time.Time
}

func (s *HealthChecker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Printf("Serving the Check request for health check, started at %v", s.started)

	if req.Service != service {
		return nil, status.Errorf(codes.NotFound, "unknown service")
	}

	duration := time.Since(s.started)
	if ((forceUnhealthy != nil) && *forceUnhealthy) || ((delayUnhealthySec >= 0) && (duration.Seconds() >= float64(delayUnhealthySec))) {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *HealthChecker) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func NewHealthChecker(started time.Time) *HealthChecker {
	return &HealthChecker{
		started: started,
	}
}

func rootmain(cmd *cobra.Command, args []string) {
	started := time.Now()

	http.HandleFunc("/make-not-serving", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Mark as unhealthy")
		forceUnhealthy = new(bool)
		*forceUnhealthy = true
		w.WriteHeader(200)
		data := (time.Since(started)).String()
		_, e := w.Write([]byte(data))
		if e != nil {
			log.Fatalf("Error from Write(): %s", e)
		}
	})

	http.HandleFunc("/make-serving", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Mark as healthy")
		forceUnhealthy = new(bool)
		*forceUnhealthy = false
		w.WriteHeader(200)
		data := (time.Since(started)).String()
		_, e := w.Write([]byte(data))
		if e != nil {
			log.Fatalf("Error from Write(): %s", e)
		}
	})

	go func() {
		httpServerAdr := fmt.Sprintf(":%d", httpPort)
		log.Printf("Http server starting to listen on %s", httpServerAdr)
		log.Fatal(http.ListenAndServe(httpServerAdr, nil))
	}()

	serverAdr := fmt.Sprintf(":%d", port)
	listenAddr, err := net.Listen("tcp", serverAdr)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Error while starting the listening service %v", err.Error()))
	}

	grpcServer := grpc.NewServer()
	healthService := NewHealthChecker(started)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthService)

	log.Printf("gRPC server starting to listen on %s", serverAdr)
	if err = grpcServer.Serve(listenAddr); err != nil {
		log.Fatalf(fmt.Sprintf("Error while starting the gRPC server on the %s listen address %v", listenAddr, err.Error()))
	}

	select {}
}

func main() {
	if err := CmdGrpcHealthChecking.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
