package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testToken = "0123456789abcdef0123456789abcdef"

func TestClientUsesLoopbackBearerAndMapsReadOnlyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			t.Fatalf("missing bridge bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/status":
			_, _ = w.Write([]byte(`{"status":"connected","version":"2.2.7"}`))
		case "/v1/projects":
			_, _ = w.Write([]byte(`{"projects":[{"id":"jarvis","name":"Jarvis"}]}`))
		case "/v1/projects/jarvis/sessions":
			_, _ = w.Write([]byte(`{"sessions":[{"id":"ses_1","title":"Mobile work","updated_at":42,"outcome":"succeeded"}]}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, testToken)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(context.Background())
	if err != nil || status.Version != "2.2.7" || status.Status != "connected" {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
	projects, err := client.Projects(context.Background())
	if err != nil || len(projects) != 1 || projects[0].ID != "jarvis" {
		t.Fatalf("projects = %#v, err = %v", projects, err)
	}
	sessions, err := client.Sessions(context.Background(), "jarvis")
	if err != nil || len(sessions) != 1 || sessions[0].Title != "Mobile work" {
		t.Fatalf("sessions = %#v, err = %v", sessions, err)
	}
}

func TestNewClientRequiresLongTokenAndLoopbackURL(t *testing.T) {
	if _, err := NewClient("http://127.0.0.1:18110", "short"); err == nil {
		t.Fatal("expected short token rejection")
	}
	if _, err := NewClient("http://example.com", testToken); err == nil {
		t.Fatal("expected non-loopback URL rejection")
	}
	if _, err := NewClient("https://127.0.0.1:18110", testToken); err == nil {
		t.Fatal("expected unsupported scheme rejection")
	}
}

func TestSessionsRejectsInvalidProjectIDsBeforeRequest(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:18110", testToken)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../secret", "foo/bar", "", strings.Repeat("a", 65)} {
		if _, err := client.Sessions(context.Background(), id); err == nil {
			t.Fatalf("expected invalid project ID rejection for %q", id)
		}
	}
}
