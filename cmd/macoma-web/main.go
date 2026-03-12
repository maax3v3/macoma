package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maax3v3/macoma/v2/internal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	maxBodyMB := flag.Int64("max-body-mb", 10, "Maximum request body size in MB")
	timeoutSec := flag.Int("timeout-sec", 30, "Request timeout in seconds")
	previewMaxDim := flag.Int("preview-max-dim", web.PreviewMaxDimension, "Maximum preview width/height in pixels")
	flag.Parse()

	cfg := web.DefaultConfig()
	cfg.MaxBodyBytes = *maxBodyMB << 20
	cfg.RequestTimeout = time.Duration(*timeoutSec) * time.Second
	cfg.PreviewMaxDimension = *previewMaxDim

	handler, err := web.Handler(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "building handler: %v\n", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("macoma-web listening on %s\n", *addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		fmt.Println("shutdown signal received")
		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "server shutdown error: %v\n", err)
			os.Exit(1)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "server stop error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("server stopped")
	}
}
