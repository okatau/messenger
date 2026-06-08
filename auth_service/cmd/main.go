package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "auth_service/internal/server/http"
	sl "auth_service/pkg/service_logger"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	srv := httpserver.New()

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("auth service stopped: %v", sl.Err(err))
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	srv.Stop(shutdownCtx)
}
