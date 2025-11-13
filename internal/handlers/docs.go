package handlers

import (
	"net/http"
	"os"
	"path/filepath"
)

func ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	// Try multiple possible paths
	paths := []string{
		"openapi.yaml",
		"./openapi.yaml",
		"../openapi.yaml",
		"../../openapi.yaml",
		"internal/static/openapi.yaml",
	}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		// Try to find file in current working directory
		wd, _ := os.Getwd()
		possiblePath := filepath.Join(wd, "openapi.yaml")
		data, err = os.ReadFile(possiblePath)
		if err != nil {
			http.Error(w, "OpenAPI spec not found", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func ServeDocs(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>API Documentation</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body {
			margin: 0;
			padding: 0;
		}
	</style>
</head>
<body>
	<script id="api-reference" data-configuration='{"url":"/api/openapi.yaml","theme":"default","darkMode":true}'></script>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@latest/dist/browser/standalone.js"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}
