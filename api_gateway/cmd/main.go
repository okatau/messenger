package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "api_gateway/internal/server/http"
)

func main() {
	srv, err := httpserver.New()
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("api gateway service stopped: %v", err)
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	srv.Stop(shutdownCtx)
}
