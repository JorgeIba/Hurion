package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/JorgeIba/Hurion/web"
)

const pidFilePath = "web/.server.pid"

func main() {
	defaultPort := 8080
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	portFlag := flag.Int("port", defaultPort, "Port to run the Web UI dev server on")
	flag.Parse()

	port := *portFlag
	webDir := "./web"
	exportsDir := os.Getenv("EXPORTS_DIR")
	if exportsDir == "" {
		exportsDir = "./data/exports"
	}

	// Ensure local directories exist for disk-backed dev server
	_ = os.MkdirAll(filepath.Clean(webDir), 0o755)
	_ = os.MkdirAll(filepath.Clean(exportsDir), 0o755)

	// Write PID file for process tracking
	pid := os.Getpid()
	_ = os.WriteFile(pidFilePath, []byte(strconv.Itoa(pid)), 0o644)
	defer func() {
		_ = os.Remove(pidFilePath)
	}()

	srv := web.NewServer(webDir, exportsDir, web.WithPort(port))
	fmt.Printf("==> Hurion Web UI server (PID %d) running at http://localhost:%d\n", pid, port)
	fmt.Printf("==> Serving web assets from %s and exports from %s\n", webDir, exportsDir)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
