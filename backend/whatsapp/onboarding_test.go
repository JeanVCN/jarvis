package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func newOnboardingRouter(t *testing.T, name string) messageRouter {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), name+".db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	router := messageRouter{db: db}
	if err := router.init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return router
}

func TestOnboardingRunsPerAccountAndPersistsPreferences(t *testing.T) {
	ctx := context.Background()
	first := newOnboardingRouter(t, "first")
	second := newOnboardingRouter(t, "second")
	if err := first.startOnboarding(ctx); err != nil {
		t.Fatal(err)
	}

	reply, handled, err := first.onboardingReply(ctx, "hello", "Oi, Jarvis", false)
	if err != nil || !handled || !strings.Contains(reply, "1/10") {
		t.Fatalf("first onboarding reply = %q, handled=%t, err=%v", reply, handled, err)
	}
	duplicate, handled, err := first.onboardingReply(ctx, "hello", "Oi, Jarvis", false)
	if err != nil || !handled || !strings.Contains(duplicate, "1/10") {
		t.Fatalf("duplicate onboarding reply = %q, handled=%t, err=%v", duplicate, handled, err)
	}

	answers := []string{
		"J.A.R.V.I.S.",
		"Marina",
		"Amigável",
		"De vez em quando",
		"Respostas curtas",
		"Resumo primeiro",
		"Humor leve",
		"Pergunte antes",
		"Português brasileiro",
		"Não use apelidos",
	}
	for i, answer := range answers {
		reply, handled, err = first.onboardingReply(ctx, "answer-"+itoa(i), answer, false)
		if err != nil || !handled {
			t.Fatalf("answer %d: handled=%t, err=%v", i+1, handled, err)
		}
		if i < len(answers)-1 && !strings.Contains(reply, itoa(i+2)+"/10") {
			t.Fatalf("answer %d reply = %q; want next question %d/10", i+1, reply, i+2)
		}
	}
	if !strings.Contains(reply, "Obrigado!") {
		t.Fatalf("completion reply = %q; want confirmation", reply)
	}
	if name := first.onboardingAssistantName(ctx, "Jarvis"); name != "J.A.R.V.I.S." {
		t.Fatalf("assistant name = %q; want onboarding answer", name)
	}

	profile, err := first.onboardingProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	persona := onboardingPersona("base persona", profile)
	for _, expected := range []string{"Marina", "Amigável", "De vez em quando", "Não use apelidos"} {
		if !strings.Contains(persona, expected) {
			t.Errorf("personalized persona %q does not include %q", persona, expected)
		}
	}
	if _, handled, err := first.onboardingReply(ctx, "after", "Oi", false); err != nil || handled {
		t.Fatalf("completed onboarding handled next message: handled=%t, err=%v", handled, err)
	}
	if _, handled, err := second.onboardingReply(ctx, "other", "Oi", false); err != nil || handled {
		t.Fatalf("unpaired account started onboarding: handled=%t, err=%v", handled, err)
	}
}

func TestOnboardingCanSkipAndRequestsTextAnswers(t *testing.T) {
	ctx := context.Background()
	router := newOnboardingRouter(t, "account")
	if err := router.startOnboarding(ctx); err != nil {
		t.Fatal(err)
	}
	if _, handled, err := router.onboardingReply(ctx, "first", "Oi", false); err != nil || !handled {
		t.Fatalf("start onboarding: handled=%t, err=%v", handled, err)
	}
	reply, handled, err := router.onboardingReply(ctx, "voice", "", true)
	if err != nil || !handled || !strings.Contains(reply, "responda por texto") || !strings.Contains(reply, "1/10") {
		t.Fatalf("media response = %q, handled=%t, err=%v", reply, handled, err)
	}
	reply, handled, err = router.onboardingReply(ctx, "skip", "pular", false)
	if err != nil || !handled || !strings.Contains(reply, "2/10") {
		t.Fatalf("skip response = %q, handled=%t, err=%v", reply, handled, err)
	}
	profile, err := router.onboardingProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := profile[0]; exists {
		t.Fatal("skipped answer was persisted")
	}
}
