package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "chat_service/internal/server/http"
	"chat_service/pkg/service_logger"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()

	srv := httpserver.New(hubCtx)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("auth service stopped: %v", service_logger.Err(err))
		}
	}()

	<-quit
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	srv.Stop(shutdownCtx)
}
