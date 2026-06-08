package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"friends_service/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.New(ctx)
	if err := srv.Start(ctx); err != nil {
		fmt.Printf("server error %v", err)
	}
}
