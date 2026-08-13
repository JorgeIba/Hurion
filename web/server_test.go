package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Handler(t *testing.T) {
	t.Parallel()

	// Setup temp directories for web and exports
	tempWebDir := t.TempDir()
	tempExportsDir := t.TempDir()

	// Write mock compiled index.html in dist/
	distDir := filepath.Join(tempWebDir, "dist")
	err := os.MkdirAll(distDir, 0o755)
	require.NoError(t, err)

	indexContent := []byte("<html><body>Hurion Web UI</body></html>")
	err = os.WriteFile(filepath.Join(distDir, "index.html"), indexContent, 0o644)
	require.NoError(t, err)

	// Write mock JSON export
	jsonContent := []byte(`{"classCode":"basicos_1"}`)
	err = os.WriteFile(filepath.Join(tempExportsDir, "basicos_1_score.json"), jsonContent, 0o644)
	require.NoError(t, err)

	srv := NewServer(tempWebDir, tempExportsDir, WithPort(8080))
	handler := srv.Handler()

	t.Run("serves root index.html from dist", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Hurion Web UI")
	})

	t.Run("serves exports JSON file", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/exports/basicos_1_score.json", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "basicos_1")
	})

	t.Run("returns 404 for non-existent export", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/exports/non_existent.json", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("configures custom options and in-memory filesystem", func(t *testing.T) {
		t.Parallel()

		mapFS := fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("In-Memory Web UI")},
		}

		optSrv := NewServer("", "", WithPort(9090), WithWebFS(mapFS))
		assert.Equal(t, 9090, optSrv.port)

		optHandler := optSrv.Handler()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		optHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "In-Memory Web UI")
	})

	t.Run("returns HTTP 503 build notice when dist directory is missing", func(t *testing.T) {
		t.Parallel()
		uncompiledWebDir := t.TempDir()

		uncompiledSrv := NewServer(uncompiledWebDir, tempExportsDir)
		uncompiledHandler := uncompiledSrv.Handler()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		uncompiledHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), "Scoreboard Frontend Not Built")
	})

	t.Run("WithWebFS option overrides fallbackHandler when dist directory is missing", func(t *testing.T) {
		t.Parallel()
		uncompiledWebDir := t.TempDir()

		customFS := fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("Custom Embedded UI")},
		}

		srv := NewServer(uncompiledWebDir, tempExportsDir, WithWebFS(customFS))
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		srv.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Custom Embedded UI")
	})
}
