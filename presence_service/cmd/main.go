package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"presence_service/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.New(ctx)
	if err := srv.Start(); err != nil {
		log.Print("presence service stopped", err)
	}
	srv.Stop()
}
