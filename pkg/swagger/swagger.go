package swagger

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"logistics/pkg/logger"
)

// Config конфигурация Swagger UI
type Config struct {
	Title                    string
	BasePath                 string
	SpecPath                 string
	DeepLinking              bool
	DocExpansion             string
	DefaultModelsExpandDepth int
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Title:                    "Logistics API",
		BasePath:                 "/swagger",
		SpecPath:                 "/openapi.json",
		DeepLinking:              true,
		DocExpansion:             "list",
		DefaultModelsExpandDepth: 1,
	}
}

// Handler HTTP handler для Swagger UI
type Handler struct {
	config   *Config
	spec     []byte
	specETag string
}

// NewHandler создаёт новый Swagger handler
func NewHandler(cfg *Config, spec []byte) *Handler {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Handler{
		config:   cfg,
		spec:     spec,
		specETag: fmt.Sprintf(`"%x"`, time.Now().UnixNano()),
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, h.config.BasePath)
	path = strings.TrimPrefix(path, "/")

	switch path {
	case "", "index.html":
		h.serveUI(w, r)
	case "openapi.json", "swagger.json", "api.json":
		h.serveSpec(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) serveUI(w http.ResponseWriter, _ *http.Request) {
	data := struct {
		Title                    string
		SpecURL                  string
		DeepLinking              bool
		DocExpansion             string
		DefaultModelsExpandDepth int
	}{
		Title:                    h.config.Title,
		SpecURL:                  h.config.BasePath + h.config.SpecPath,
		DeepLinking:              h.config.DeepLinking,
		DocExpansion:             h.config.DocExpansion,
		DefaultModelsExpandDepth: h.config.DefaultModelsExpandDepth,
	}

	tmpl, err := template.New("swagger-ui").Parse(swaggerUITemplate)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if err := tmpl.Execute(w, data); err != nil {
		logger.Log.Error("Failed to execute swagger template", "error", err)
	}
}

func (h *Handler) serveSpec(w http.ResponseWriter, r *http.Request) {
	if match := r.Header.Get("If-None-Match"); match == h.specETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("ETag", h.specETag)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if _, err := w.Write(h.spec); err != nil {
		logger.Log.Error("Failed to write spec", "error", err)
	}
}

const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5/favicon-32x32.png" sizes="32x32">
    <style>
        html { box-sizing: border-box; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; padding: 0; background: #fafafa; }
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
        .swagger-ui .info .title { font-size: 36px; color: #3b4151; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" charset="UTF-8"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "{{.SpecURL}}",
                dom_id: '#swagger-ui',
                deepLinking: {{.DeepLinking}},
                docExpansion: "{{.DocExpansion}}",
                defaultModelsExpandDepth: {{.DefaultModelsExpandDepth}},
                presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
                plugins: [SwaggerUIBundle.plugins.DownloadUrl],
                layout: "StandaloneLayout",
                validatorUrl: null
            });
        };
    </script>
</body>
</html>`

// Server HTTP сервер для Swagger UI
type Server struct {
	config     *Config
	spec       []byte
	httpServer *http.Server
}

// NewServer создаёт новый Swagger сервер
func NewServer(cfg *Config, spec []byte) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Server{
		config: cfg,
		spec:   spec,
	}
}

// Start запускает сервер
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	handler := NewHandler(s.config, s.spec)
	mux.Handle(s.config.BasePath+"/", handler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, s.config.BasePath+"/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			logger.Log.Debug("Failed to write health response", "error", err)
		}
	})

	addr := fmt.Sprintf(":%d", port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Log.Info("Starting Swagger UI server", "addr", addr, "path", s.config.BasePath)
	return s.httpServer.ListenAndServe()
}

// Shutdown останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// RegisterRoutes регистрирует маршруты в существующем mux
func RegisterRoutes(mux *http.ServeMux, cfg *Config, spec []byte) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	handler := NewHandler(cfg, spec)
	mux.Handle(cfg.BasePath+"/", handler)
}
