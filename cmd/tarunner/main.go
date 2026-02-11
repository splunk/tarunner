// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/splunk/tarunner/internal/collector"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <basedir> <otlp-endpoint>", os.Args[0])
	}
	basedir := os.Args[1]
	otlpEndpoint := os.Args[2]

	shutdownFunc, err := collector.Run(basedir, otlpEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	if shutdownFunc == nil {
		return
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	waitShutdown := make(chan struct{})
	go func() {
		<-signalChan
		shutdownFunc()
		close(waitShutdown)
	}()
	<-waitShutdown
}
