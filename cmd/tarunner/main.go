package main

import (
	"context"
	"github.com/splunk/tarunner/internal/exporter"
	"go.opentelemetry.io/collector/component/componenttest"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/process"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <basedir> <otlp-endpoint>", os.Args[0])
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	e, err := exporter.NewExporter(logger, os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	err = e.Start(context.Background(), componenttest.NewNopHost())
	if err != nil {
		log.Fatal(err)
	}
	runner := process.NewRunner(e, logger)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		runner.Shutdown()
		_ = e.Shutdown(context.Background())
	}()

	scheduling, err := runner.Run(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	if !scheduling {
		// No jobs to schedule. Exit.
		os.Exit(0)
	}

	<-runner.Done()

}
