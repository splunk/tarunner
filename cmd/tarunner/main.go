// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/splunk/tarunner/internal/config"

	"github.com/splunk/tarunner/internal/collector"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <basedir>", os.Args[0])
	}
	basedir := os.Args[1]
	configFile := filepath.Join(basedir, "tarunner.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatalf("config file %q does not exist", configFile)
	}
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	shutdownFunc, err := collector.Run(basedir, cfg)
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
