// services/gateway-svc/internal/middleware/cors_test.go

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"logistics/pkg/config"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		cfg            config.CORSConfig
		requestOrigin  string
		requestMethod  string
		expectedOrigin string
		expectNoOrigin bool
	}{
		{
			name: "allowed origin",
			cfg: config.CORSConfig{
				AllowedOrigins:   []string{"http://localhost:3000"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
			},
			requestOrigin:  "http://localhost:3000",
			requestMethod:  "GET",
			expectedOrigin: "http://localhost:3000",
		},
		{
			name: "wildcard origin",
			cfg: config.CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Content-Type"},
			},
			requestOrigin:  "http://any-origin.com",
			requestMethod:  "GET",
			expectedOrigin: "http://any-origin.com",
		},
		{
			name: "not allowed origin",
			cfg: config.CORSConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Content-Type"},
			},
			requestOrigin:  "http://evil.com",
			requestMethod:  "GET",
			expectNoOrigin: true,
		},
		{
			name: "preflight request",
			cfg: config.CORSConfig{
				AllowedOrigins:   []string{"http://localhost:3000"},
				AllowedMethods:   []string{"GET", "POST", "PUT"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
			},
			requestOrigin:  "http://localhost:3000",
			requestMethod:  "OPTIONS",
			expectedOrigin: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			corsHandler := CORS(tt.cfg)(nextHandler)

			// Create request
			req := httptest.NewRequest(tt.requestMethod, "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			corsHandler.ServeHTTP(rr, req)

			// Check headers
			origin := rr.Header().Get("Access-Control-Allow-Origin")

			if tt.expectNoOrigin {
				if origin != "" {
					t.Errorf("Expected no origin header, got %v", origin)
				}
			} else {
				if origin != tt.expectedOrigin {
					t.Errorf("Access-Control-Allow-Origin = %v, want %v", origin, tt.expectedOrigin)
				}
			}

			// Check preflight response
			if tt.requestMethod == "OPTIONS" {
				if rr.Code != http.StatusNoContent {
					t.Errorf("Preflight response code = %d, want %d", rr.Code, http.StatusNoContent)
				}
				maxAge := rr.Header().Get("Access-Control-Max-Age")
				if maxAge != "86400" {
					t.Errorf("Access-Control-Max-Age = %v, want 86400", maxAge)
				}
			}

			// Check credentials header
			if tt.cfg.AllowCredentials && !tt.expectNoOrigin {
				creds := rr.Header().Get("Access-Control-Allow-Credentials")
				if creds != "true" {
					t.Errorf("Access-Control-Allow-Credentials = %v, want true", creds)
				}
			}
		})
	}
}

func TestCORS_MethodsAndHeaders(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Custom-Header"},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CORS(cfg)(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost")

	rr := httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	methods := rr.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("Access-Control-Allow-Methods should be set")
	}

	headers := rr.Header().Get("Access-Control-Allow-Headers")
	if headers == "" {
		t.Error("Access-Control-Allow-Headers should be set")
	}
}
