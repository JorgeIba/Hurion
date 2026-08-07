package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"
)

// Option defines a functional option for configuring Server.
type Option func(*Server)

// WithPort configures the TCP listening port.
func WithPort(port int) Option {
	return func(s *Server) {
		if port > 0 {
			s.port = port
		}
	}
}

// WithWebFS configures a custom web filesystem for UI assets (e.g. embed.FS or fstest.MapFS).
func WithWebFS(webFS fs.FS) Option {
	return func(s *Server) {
		if webFS != nil {
			s.webFS = webFS
		}
	}
}

// WithExportsFS configures a custom exports filesystem for JSON scores.
func WithExportsFS(exportsFS fs.FS) Option {
	return func(s *Server) {
		if exportsFS != nil {
			s.exportsFS = exportsFS
		}
	}
}

// Server manages the HTTP static file web server.
type Server struct {
	port            int
	webFS           fs.FS
	exportsFS       fs.FS
	shutdownTimeout time.Duration
}

// NewServer constructs a Server with sensible defaults and optional functional configuration.
func NewServer(webDir, exportsDir string, opts ...Option) *Server {
	s := &Server{
		port:            8080,
		shutdownTimeout: 3 * time.Second,
	}

	if webDir != "" {
		s.webFS = os.DirFS(webDir)
	}
	if exportsDir != "" {
		s.exportsFS = os.DirFS(exportsDir)
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Handler constructs the HTTP routing handler cleanly without side effects.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 1. Route for Scoreboard JSON exports (/exports/<classCode>_score.json)
	// http.StripPrefix returns a wrapper handler that cuts off "/exports/" from incoming request URLs
	// (e.g. "/exports/basicos_1_score.json" -> "basicos_1_score.json") and then delegates the modified request
	// to http.FileServer, which matches whatever filename remains against the files inside s.exportsFS.
	if s.exportsFS != nil {
		exportsHandler := http.StripPrefix("/exports/", http.FileServer(http.FS(s.exportsFS)))
		mux.Handle("GET /exports/", exportsHandler)
	}

	// 2. Route for Web UI Frontend static assets (index.html, app.js, styles.css)
	if s.webFS != nil {
		mux.Handle("GET /", http.FileServer(http.FS(s.webFS)))
	}

	return mux
}

// Start launches the HTTP server and blocks until ctx is canceled.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.Handler(),
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
