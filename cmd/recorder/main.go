package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"recorder/pkg/config"
	"recorder/pkg/worker"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")

	cfgPath := flag.String("config", "config.yaml", "path to YAML configuration file")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	cams, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown signal received, stopping recordings...")
		cancel()
	}()

	for _, cam := range cams {
		go worker.Run(ctx, cam)
	}

	<-ctx.Done()
	log.Println("all workers stopped")
}
