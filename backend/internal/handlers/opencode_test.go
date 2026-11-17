package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/opencode"
)

func TestOpenCodeProjectsRouteDoesNotExposeDirectories(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer 0123456789abcdef0123456789abcdef" {
			t.Fatalf("bridge request missing authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"projects":[{"id":"jarvis","name":"Jarvis"}]}`))
	}))
	defer bridge.Close()

	client, err := opencode.NewClient(bridge.URL, "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterOpenCode(router, client, opencode.Store{}, false)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/opencode/projects", nil)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"name":"Jarvis"`) {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "directory") || strings.Contains(response.Body.String(), "/home/") {
		t.Fatalf("response leaked a filesystem path: %s", response.Body.String())
	}
}

func TestOpenCodeWritesStayDisabledUnlessExplicitlyEnabled(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("disabled write reached bridge: %s %s", r.Method, r.URL.Path)
	}))
	defer bridge.Close()
	client, err := opencode.NewClient(bridge.URL, "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterOpenCode(router, client, opencode.Store{}, false)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/opencode/projects/jarvis/sessions", strings.NewReader(`{"operation_id":"0123456789abcdef0123456789abcdef"}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "opencode_write_disabled") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
}
