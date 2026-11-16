package opencode

import (
	"context"
	"encoding/json"
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

func TestClientPostsOnlyBoundedProjectAndOperationInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			t.Fatalf("missing bridge bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		body := map[string]string{}
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
		}
		switch r.URL.Path {
		case "/v1/projects/jarvis/sessions":
			if r.Method != http.MethodPost || body["operation_id"] != "0123456789abcdef0123456789abcdef" {
				t.Fatalf("unexpected create request: %s %#v", r.Method, body)
			}
			_, _ = w.Write([]byte(`{"id":"ses_created","title":"Jarvis","updated_at":42}`))
		case "/v1/projects/jarvis/sessions/ses_created/prompts":
			if r.Method != http.MethodPost || body["text"] != "safe task" {
				t.Fatalf("unexpected prompt request: %s %#v", r.Method, body)
			}
			_, _ = w.Write([]byte(`{"inbox_id":"msg_1"}`))
		case "/v1/projects/jarvis/operations/0123456789abcdef0123456789abcdef":
			_, _ = w.Write([]byte(`{"session":{"id":"ses_created","title":"Jarvis","updated_at":42}}`))
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
	ctx := context.Background()
	created, err := client.CreateSession(ctx, "jarvis", "0123456789abcdef0123456789abcdef")
	if err != nil || created.ID != "ses_created" {
		t.Fatalf("create = %#v, err=%v", created, err)
	}
	accepted, err := client.Prompt(ctx, "jarvis", created.ID, "0123456789abcdef0123456789abcdef", "safe task")
	if err != nil || accepted.InboxID != "msg_1" {
		t.Fatalf("prompt = %#v, err=%v", accepted, err)
	}
	found, err := client.FindCreatedSession(ctx, "jarvis", "0123456789abcdef0123456789abcdef")
	if err != nil || found == nil || found.ID != created.ID {
		t.Fatalf("reconcile = %#v, err=%v", found, err)
	}
	if _, err := client.Prompt(ctx, "jarvis", created.ID, "../bad", "safe task"); err == nil {
		t.Fatal("expected invalid operation id rejection")
	}
}
