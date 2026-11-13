package conversation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPStreamReassemblesToolArguments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Olá.\"}}]}\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call1","function":{"name":"calculator","arguments":"{\"input\":"}}]}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"25 * 4\"}"}}]}}]}`+"\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	var text strings.Builder
	calls, err := (HTTPProvider{URL: server.URL, Client: server.Client()}).Stream(context.Background(), []Message{{Role: "user", Content: "calcule"}}, func(s string) error { text.WriteString(s); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "Olá." || len(calls) != 1 || calls[0].Function.Arguments != `{"input":"25 * 4"}` {
		t.Fatalf("invalid stream: %q %+v", text.String(), calls)
	}
}
func TestQuotaAndTruncatedStream(t *testing.T) {
	for _, status := range []int{429, 401, 404, 200} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(status)
			}))
			defer server.Close()
			_, e := (HTTPProvider{URL: server.URL}).Stream(context.Background(), nil, func(string) error { return nil })
			if e == nil {
				t.Fatal("expected failure")
			}
			if status == 429 {
				var f *Failure
				if !errors.As(e, &f) || f.Code != "quota_exhausted" || f.RetryAfter != "30" {
					t.Fatal(e)
				}
			}
		})
	}
}
func TestScriptedCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	chunks := 0
	_, e := (Scripted{Delay: time.Millisecond}).Stream(ctx, []Message{{Role: "user", Content: "longa"}}, func(string) error { chunks++; cancel(); return nil })
	if !errors.Is(e, context.Canceled) || chunks != 1 {
		t.Fatalf("chunks=%d err=%v", chunks, e)
	}
}
func TestProviderActivationNeverFallsBack(t *testing.T) {
	t.Setenv("JARVIS_GROQ_FREE_CONFIRMED", "")
	t.Setenv("JARVIS_CODEX_ENABLED", "")
	for _, name := range []string{"groq", "codex", "unknown"} {
		p, e := Resolve(name, "")
		if e == nil || p != nil {
			t.Fatalf("unexpected activation %s", name)
		}
	}
}
