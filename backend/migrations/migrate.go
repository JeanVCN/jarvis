// Package migrations applies additive migrations to existing and fresh databases.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
)

//go:embed *.sql
var files embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(74206113)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); e != nil {
		return e
	}
	names, e := fs.Glob(files, "*.sql")
	if e != nil {
		return e
	}
	for _, name := range names {
		var done bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", name).Scan(&done); e != nil {
			return e
		}
		if done {
			continue
		}
		b, e := files.ReadFile(name)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, string(b)); e != nil {
			return fmt.Errorf("migration %s: %w", name, e)
		}
		if _, e = tx.Exec(ctx, "INSERT INTO schema_migrations(name) VALUES($1)", name); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, `WITH recovered AS (UPDATE conversation_turns SET status='interrupted',finished_at=now(),generated=COALESCE((SELECT string_agg(e.data->>'delta','' ORDER BY e.seq) FROM conversation_events e WHERE e.turn_id=conversation_turns.id AND e.type='text.delta'),'') WHERE status='running' RETURNING id) INSERT INTO conversation_events(turn_id,type,data) SELECT id,'turn.finished','{"status":"interrupted","reason":"server_restart"}'::jsonb FROM recovered`); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE conversation_tools SET status='uncertain' WHERE status='claimed'"); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
