package grpckit

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strings"
)

// swaggerUIHTML is the HTML template for Swagger UI.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "{{.SpecURL}}",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout"
            });
        };
    </script>
</body>
</html>`

// swaggerHandler manages Swagger UI and spec serving.
type swaggerHandler struct {
	specPath string
	specData []byte
}

// newSwaggerHandler creates a new Swagger handler from a file path.
func newSwaggerHandler(specPath string) (*swaggerHandler, error) {
	// Read the OpenAPI spec file
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	// Validate it's valid JSON
	var js json.RawMessage
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, err
	}

	return &swaggerHandler{
		specPath: specPath,
		specData: data,
	}, nil
}

// newSwaggerHandlerFromBytes creates a new Swagger handler from embedded data.
func newSwaggerHandlerFromBytes(data []byte) (*swaggerHandler, error) {
	// Validate it's valid JSON
	var js json.RawMessage
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, err
	}

	return &swaggerHandler{
		specData: data,
	}, nil
}

// UIHandler returns the Swagger UI HTML page handler.
func (s *swaggerHandler) UIHandler() http.HandlerFunc {
	tmpl := template.Must(template.New("swagger").Parse(swaggerUIHTML))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		data := struct {
			SpecURL string
		}{
			SpecURL: "/swagger/spec.json",
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render Swagger UI", http.StatusInternalServerError)
		}
	}
}

// SpecHandler returns the OpenAPI spec JSON handler.
func (s *swaggerHandler) SpecHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(s.specData)
	}
}

// registerSwaggerEndpoints registers Swagger endpoints on the mux from a file path.
func registerSwaggerEndpoints(mux *http.ServeMux, specPath string) error {
	handler, err := newSwaggerHandler(specPath)
	if err != nil {
		return err
	}

	registerSwaggerHandler(mux, handler)
	return nil
}

// registerSwaggerEndpointsFromBytes registers Swagger endpoints from embedded data.
func registerSwaggerEndpointsFromBytes(mux *http.ServeMux, data []byte) error {
	handler, err := newSwaggerHandlerFromBytes(data)
	if err != nil {
		return err
	}

	registerSwaggerHandler(mux, handler)
	return nil
}

// registerSwaggerHandler registers the swagger handler on the mux.
func registerSwaggerHandler(mux *http.ServeMux, handler *swaggerHandler) {
	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/swagger")
		if path == "" || path == "/" {
			handler.UIHandler()(w, r)
			return
		}
		if path == "/spec.json" {
			handler.SpecHandler()(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// registerSwaggerNotFound registers a 404 handler for swagger endpoints.
// This is used when swagger is enabled but no data was loaded (make swagger wasn't run).
func registerSwaggerNotFound(mux *http.ServeMux) {
	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "swagger not available - run 'make swagger' to enable", http.StatusNotFound)
	})
}
