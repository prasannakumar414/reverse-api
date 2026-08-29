package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsEndpointServesSwaggerUI(t *testing.T) {
	server := NewServer(Config{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs", nil)
	server.server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response code = %d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	body := response.Body.String()
	if !strings.Contains(body, "SwaggerUIBundle") {
		t.Fatal("docs response does not include Swagger UI")
	}
	if !strings.Contains(body, `url: "/docs/openapi.yaml"`) {
		t.Fatal("docs response does not reference /docs/openapi.yaml")
	}
}

func TestDocsSlashEndpointServesSwaggerUI(t *testing.T) {
	server := NewServer(Config{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	server.server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response code = %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Profile Retrieval API Docs") {
		t.Fatal("docs slash response does not include docs title")
	}
}

func TestOpenAPIEndpointServesSpec(t *testing.T) {
	server := NewServer(Config{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	server.server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response code = %d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/yaml") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"openapi: 3.0.3",
		"  /healthz:",
		"  /profiles/retrieve:",
		"    ProfileResult:",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("openapi response missing %q", expected)
		}
	}
}

func TestDocsEndpointRejectsNonGET(t *testing.T) {
	server := NewServer(Config{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/docs", nil)
	server.server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("response code = %d", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q", allow)
	}
}
