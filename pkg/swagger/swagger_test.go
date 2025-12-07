package swagger

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Title == "" {
		t.Error("Title should not be empty")
	}
	if cfg.BasePath == "" {
		t.Error("BasePath should not be empty")
	}
	if cfg.SpecPath == "" {
		t.Error("SpecPath should not be empty")
	}
}

func TestHandler_ServeHTTP_UI(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.0"}`)
	handler := NewHandler(nil, spec)

	tests := []struct {
		path string
	}{
		{"/swagger/"},
		{"/swagger/index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %s, want text/html; charset=utf-8", contentType)
			}

			body := w.Body.String()
			if len(body) == 0 {
				t.Error("response body should not be empty")
			}
		})
	}
}

func TestHandler_ServeHTTP_Spec(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.0","info":{"title":"Test"}}`)
	handler := NewHandler(nil, spec)

	specPaths := []string{
		"/swagger/openapi.json",
		"/swagger/swagger.json",
		"/swagger/api.json",
	}

	for _, path := range specPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %s, want application/json; charset=utf-8", contentType)
			}

			body := w.Body.Bytes()
			if string(body) != string(spec) {
				t.Error("response should match spec")
			}

			// Check ETag header
			etag := w.Header().Get("ETag")
			if etag == "" {
				t.Error("ETag header should be set")
			}
		})
	}
}

func TestHandler_ServeHTTP_NotFound(t *testing.T) {
	spec := []byte(`{}`)
	handler := NewHandler(nil, spec)

	req := httptest.NewRequest("GET", "/swagger/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ServeHTTP_ETagCaching(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.0"}`)
	handler := NewHandler(nil, spec)

	// First request to get ETag
	req1 := httptest.NewRequest("GET", "/swagger/openapi.json", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	etag := w1.Header().Get("ETag")

	// Second request with If-None-Match
	req2 := httptest.NewRequest("GET", "/swagger/openapi.json", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d (Not Modified)", w2.Code, http.StatusNotModified)
	}
}

func TestHandler_CustomConfig(t *testing.T) {
	cfg := &Config{
		Title:                    "Custom API",
		BasePath:                 "/api-docs",
		SpecPath:                 "/spec.json",
		DeepLinking:              false,
		DocExpansion:             "none",
		DefaultModelsExpandDepth: 0,
	}
	spec := []byte(`{}`)
	handler := NewHandler(cfg, spec)

	req := httptest.NewRequest("GET", "/api-docs/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !containsString(body, "Custom API") {
		t.Error("response should contain custom title")
	}
}

func TestNewServer(t *testing.T) {
	server := NewServer(nil, []byte(`{}`))

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.config == nil {
		t.Error("server.config should not be nil")
	}
}

func TestRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	spec := []byte(`{"openapi":"3.0.0"}`)
	cfg := &Config{
		BasePath: "/swagger",
		SpecPath: "/openapi.json",
	}

	RegisterRoutes(mux, cfg, spec)

	// Test that routes are registered
	req := httptest.NewRequest("GET", "/swagger/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("registered route status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestServer_Shutdown(t *testing.T) {
	server := NewServer(nil, []byte(`{}`))

	// Shutdown without Start should not error
	err := server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestServer_Start(t *testing.T) {
	server := NewServer(&Config{
		Title:    "Test API",
		BasePath: "/swagger",
	}, []byte(`{"openapi":"3.0.0"}`))

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// We can't easily test Start() as it blocks
	// Instead we verify the server struct is properly configured
	if server.config.Title != "Test API" {
		t.Error("server config not properly set")
	}
}

func TestHandler_CORS(t *testing.T) {
	spec := []byte(`{}`)
	handler := NewHandler(nil, spec)

	req := httptest.NewRequest("GET", "/swagger/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	cors := w.Header().Get("Access-Control-Allow-Origin")
	if cors != "*" {
		t.Errorf("CORS header = %s, want *", cors)
	}
}

func BenchmarkHandler_ServeSpec(b *testing.B) {
	spec := make([]byte, 100000) // 100KB spec
	handler := NewHandler(nil, spec)

	req := httptest.NewRequest("GET", "/swagger/openapi.json", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		io.Copy(io.Discard, w.Body)
	}
}
