package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	mode := flag.String("mode", "client", "Mode: 'host' for central-server host or 'client' for central-server client")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := NewApp(*mode)
	if err := app.Start(ctx); err != nil {
		log.Fatalf("failed to start application: %v", err)
	}

	<-ctx.Done()
}
