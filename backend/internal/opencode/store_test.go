package opencode

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jean/jarvis/backend/migrations"
)

func TestStorePersistsAndDeduplicatesOperations(t *testing.T) {
	databaseURL := os.Getenv("JARVIS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set JARVIS_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}

	store := Store{Pool: pool}
	sessionID := "11111111111111111111111111111111"
	promptID := "22222222222222222222222222222222"
	permissionID := "33333333333333333333333333333333"

	created, isNew, err := store.StartSession(ctx, sessionID, "jarvis-test")
	if err != nil || !isNew || created.State != "submitting" {
		t.Fatalf("StartSession = %#v, new=%v, err=%v", created, isNew, err)
	}
	duplicate, isNew, err := store.StartSession(ctx, sessionID, "jarvis-test")
	if err != nil || isNew || duplicate.ID != sessionID {
		t.Fatalf("duplicate StartSession = %#v, new=%v, err=%v", duplicate, isNew, err)
	}
	if _, _, err := store.StartSession(ctx, sessionID, "another-project"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflicting operation ID, got %v", err)
	}
	if err := store.AcceptSession(ctx, sessionID, "ses_test_store", "Store test"); err != nil {
		t.Fatal(err)
	}

	prompt, isNew, err := store.StartPrompt(ctx, promptID, sessionID, "inspect this project")
	if err != nil || !isNew || prompt.State != "submitting" {
		t.Fatalf("StartPrompt = %#v, new=%v, err=%v", prompt, isNew, err)
	}
	duplicate, isNew, err = store.StartPrompt(ctx, promptID, sessionID, "inspect this project")
	if err != nil || isNew || duplicate.ID != promptID {
		t.Fatalf("duplicate StartPrompt = %#v, new=%v, err=%v", duplicate, isNew, err)
	}
	if _, _, err := store.StartPrompt(ctx, promptID, sessionID, "different prompt"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected prompt conflict, got %v", err)
	}
	if err := store.AcceptPrompt(ctx, promptID, "msg_test_store"); err != nil {
		t.Fatal(err)
	}
	accepted, err := store.Operation(ctx, promptID)
	if err != nil || accepted.State != "accepted" || accepted.ExternalID == nil || *accepted.ExternalID != "msg_test_store" {
		t.Fatalf("accepted operation = %#v, err=%v", accepted, err)
	}

	permission, isNew, err := store.StartPermission(ctx, permissionID, sessionID, "per_test_store", "once")
	if err != nil || !isNew || permission.Kind != "permission" {
		t.Fatalf("StartPermission = %#v, new=%v, err=%v", permission, isNew, err)
	}
	if err := store.MarkUncertain(ctx, permissionID); err != nil {
		t.Fatal(err)
	}
	uncertain, err := store.Operation(ctx, permissionID)
	if err != nil || uncertain.State != "uncertain" {
		t.Fatalf("uncertain operation = %#v, err=%v", uncertain, err)
	}
}
