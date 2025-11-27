package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/markussiebert/homeddns/internal/auth"
	"github.com/markussiebert/homeddns/internal/handler"
	"github.com/markussiebert/homeddns/internal/provider"
)

func RunServer(port int, config *Config) error {
	prov, ok := provider.GetFactory(config.Provider)
	if !ok {
		return fmt.Errorf("provider factory not found: %s", config.Provider)
	}

	// Provider handles its own credential loading
	p, err := prov(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	log.Printf("Using DNS provider: %s", p.Name())

	dyndnsHandler := handler.NewDynDNSHandler(handler.Config{
		Provider:   p,
		DefaultTTL: config.DefaultTTL,
	})

	authMiddleware := auth.Middleware(auth.Config{
		Username:     config.Username,
		PasswordHash: config.PasswordHash,
	})

	mux := http.NewServeMux()
	mux.Handle("/", authMiddleware(dyndnsHandler))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting DynDNS server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.Close(shutdownCtx); err != nil {
		log.Printf("Warning: error closing provider: %v", err)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
