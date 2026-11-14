package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/services"
	"github.com/jean/jarvis/backend/internal/tools"
)

func TestToolHandlerList(t *testing.T) {
	t.Parallel()

	router := newToolTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tools", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"name":"calculator"`) {
		t.Fatalf("expected calculator in response, got %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"name":"time"`) {
		t.Fatalf("expected time in response, got %s", res.Body.String())
	}
}

func TestToolHandlerExecute(t *testing.T) {
	t.Parallel()

	router := newToolTestRouter(t)
	body := strings.NewReader(`{"tool":"calculator","input":"25 * 4"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tools/execute", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	if strings.TrimSpace(res.Body.String()) != `{"result":"100"}` {
		t.Fatalf("unexpected response: %s", res.Body.String())
	}
}

func newToolTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	service := newToolTestService(t)
	handler := NewToolHandler(service)

	router := gin.New()
	router.GET("/api/tools", handler.List)
	router.POST("/api/tools/execute", handler.Execute)
	return router
}

func newToolTestService(t *testing.T) *services.ToolService {
	t.Helper()

	registry := tools.NewRegistry()
	if err := registry.Register(tools.NewCalculatorTool()); err != nil {
		t.Fatalf("register calculator: %v", err)
	}
	if err := registry.Register(tools.NewTimeTool()); err != nil {
		t.Fatalf("register time: %v", err)
	}

	return services.NewToolService(registry)
}
