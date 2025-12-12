package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"logistics/pkg/config"
)

// CORS middleware для ConnectRPC
func CORS(cfg config.CORSConfig) func(http.Handler) http.Handler {
	// Предварительно подготавливаем заголовки
	allowedHeaders := prepareAllowedHeaders(cfg.AllowedHeaders)
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := fmt.Sprintf("%d", cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Проверяем origin
			allowed := false
			allowedOrigin := ""
			for _, o := range cfg.AllowedOrigins {
				if o == "*" {
					allowed = true
					allowedOrigin = "*"
					break
				}
				if o == origin {
					allowed = true
					allowedOrigin = origin
					break
				}
			}

			if allowed && allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

			if exposedHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
			}

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Preflight
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Max-Age", maxAge)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// prepareAllowedHeaders обрабатывает wildcard и добавляет обязательные заголовки
func prepareAllowedHeaders(headers []string) string {
	// Если указан wildcard, раскрываем его в конкретный список
	// потому что браузеры не включают Authorization при "*"
	for _, h := range headers {
		if h == "*" {
			return strings.Join([]string{
				"Accept",
				"Accept-Language",
				"Content-Language",
				"Content-Type",
				"Authorization",
				"Origin",
				"X-Requested-With",
				"X-Grpc-Web",
				"Grpc-Timeout",
				"Grpc-Metadata-*",
				"X-User-Agent",
				"X-Custom-Header",
			}, ", ")
		}
	}

	// Проверяем, что Authorization включён
	hasAuth := false
	for _, h := range headers {
		if strings.EqualFold(h, "Authorization") {
			hasAuth = true
			break
		}
	}

	if !hasAuth {
		headers = append(headers, "Authorization")
	}

	return strings.Join(headers, ", ")
}
