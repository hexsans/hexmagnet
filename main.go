package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hexsans/hexmagnet/internal/app"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "hexmagnet panicked: %v\n", r)
			_, _ = fmt.Fprintf(os.Stderr, "hexmagnet exiting (code=1)\n")

			os.Exit(1)
		}
	}()

	configPath := flag.String("c", "", "config file path")

	flag.Parse()

	if *configPath != "" {
		_ = os.Setenv("HEXMAGNET_CONFIG_FILE", *configPath)
	}

	_, _ = fmt.Fprintf(os.Stderr, "hexmagnet starting\n")

	a := app.New()

	if err := a.Start(context.Background()); err != nil {
		fatalf("hexmagnet startup failed: %v\n", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sig:
	case <-a.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := a.Stop(shutdownCtx); err != nil {
		fatalf("hexmagnet shutdown error: %v\n", err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "hexmagnet exited normally (code=0)\n")
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	_, _ = fmt.Fprintf(os.Stderr, "hexmagnet exiting (code=1)\n")

	//nolint:revive // exit code via helper keeps the recover-defer intact
	os.Exit(1)
}
