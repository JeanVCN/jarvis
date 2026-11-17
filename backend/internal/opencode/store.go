package opencode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Operation struct {
	ID         string  `json:"id"`
	ProjectID  string  `json:"project_id"`
	SessionID  string  `json:"session_id"`
	Kind       string  `json:"kind"`
	Input      string  `json:"input,omitempty"`
	State      string  `json:"state"`
	ExternalID *string `json:"external_id,omitempty"`
	ErrorCode  *string `json:"error_code,omitempty"`
}

type SessionLink struct {
	ID         string  `json:"id"`
	ProjectID  string  `json:"project_id"`
	ExternalID *string `json:"external_id,omitempty"`
	Title      string  `json:"title"`
	State      string  `json:"state"`
	UpdatedAt  int64   `json:"updated_at"`
}

type Store struct{ Pool *pgxpool.Pool }

var ErrConflict = errors.New("opencode_operation_conflict")
var ErrSessionNotFound = errors.New("opencode_session_not_found")
var ErrSessionUnavailable = errors.New("opencode_session_unavailable")

func (s Store) StartSession(ctx context.Context, operationID, projectID string) (Operation, bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Operation{}, false, err
	}
	defer tx.Rollback(ctx)

	if op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID)); err == nil {
		if op.Kind != "create_session" || op.ProjectID != projectID || op.SessionID != operationID {
			return Operation{}, false, ErrConflict
		}
		return op, false, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO opencode_sessions(id,project_id,state)
		VALUES($1,$2,'creating')`, operationID, projectID); err != nil {
		return Operation{}, false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO opencode_operations(id,project_id,session_id,kind,state)
		VALUES($1,$2,$1,'create_session','submitting')`, operationID, projectID); err != nil {
		return Operation{}, false, err
	}
	op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID))
	if err != nil {
		return Operation{}, false, err
	}
	return op, true, tx.Commit(ctx)
}

func (s Store) StartPrompt(ctx context.Context, operationID, sessionID, text string) (Operation, bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Operation{}, false, err
	}
	defer tx.Rollback(ctx)

	if op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID)); err == nil {
		if op.Kind != "prompt" || op.SessionID != sessionID || op.Input != text {
			return Operation{}, false, ErrConflict
		}
		return op, false, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, err
	}

	var projectID, sessionState string
	var externalID *string
	err = tx.QueryRow(ctx, `SELECT project_id,external_id,state FROM opencode_sessions WHERE id=$1 FOR UPDATE`, sessionID).
		Scan(&projectID, &externalID, &sessionState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, ErrSessionNotFound
	}
	if err != nil {
		return Operation{}, false, err
	}
	if externalID == nil || sessionState != "ready" {
		return Operation{}, false, ErrSessionUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO opencode_operations(id,project_id,session_id,kind,input,state)
		VALUES($1,$2,$3,'prompt',$4,'submitting')`, operationID, projectID, sessionID, text); err != nil {
		return Operation{}, false, err
	}
	op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID))
	if err != nil {
		return Operation{}, false, err
	}
	return op, true, tx.Commit(ctx)
}

func (s Store) StartPermission(ctx context.Context, operationID, sessionID, requestID, decision string) (Operation, bool, error) {
	if decision != "once" && decision != "reject" {
		return Operation{}, false, ErrConflict
	}
	input := "permission:" + requestID + ":" + decision
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Operation{}, false, err
	}
	defer tx.Rollback(ctx)
	if op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID)); err == nil {
		if op.Kind != "permission" || op.SessionID != sessionID || op.Input != input {
			return Operation{}, false, ErrConflict
		}
		return op, false, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, err
	}
	var projectID, sessionState string
	err = tx.QueryRow(ctx, `SELECT project_id,state FROM opencode_sessions WHERE id=$1 FOR UPDATE`, sessionID).
		Scan(&projectID, &sessionState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, ErrSessionNotFound
	}
	if err != nil {
		return Operation{}, false, err
	}
	if sessionState != "ready" {
		return Operation{}, false, ErrSessionUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO opencode_operations(id,project_id,session_id,kind,input,state,external_id)
		VALUES($1,$2,$3,'permission',$4,'submitting',$5)`, operationID, projectID, sessionID, input, requestID); err != nil {
		return Operation{}, false, err
	}
	op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID))
	if err != nil {
		return Operation{}, false, err
	}
	return op, true, tx.Commit(ctx)
}

func (s Store) StartForm(ctx context.Context, operationID, sessionID, formID string, answer map[string]any) (Operation, bool, error) {
	encoded, err := json.Marshal(answer)
	if err != nil || len(encoded) == 0 {
		return Operation{}, false, ErrConflict
	}
	digest := sha256.Sum256(encoded)
	input := "form:" + formID + ":" + hex.EncodeToString(digest[:])
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Operation{}, false, err
	}
	defer tx.Rollback(ctx)
	if op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID)); err == nil {
		if op.Kind != "form" || op.SessionID != sessionID || op.Input != input {
			return Operation{}, false, ErrConflict
		}
		return op, false, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, err
	}
	var projectID, sessionState string
	err = tx.QueryRow(ctx, `SELECT project_id,state FROM opencode_sessions WHERE id=$1 FOR UPDATE`, sessionID).
		Scan(&projectID, &sessionState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, ErrSessionNotFound
	}
	if err != nil {
		return Operation{}, false, err
	}
	if sessionState != "ready" {
		return Operation{}, false, ErrSessionUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO opencode_operations(id,project_id,session_id,kind,input,state,external_id)
		VALUES($1,$2,$3,'form',$4,'submitting',$5)`, operationID, projectID, sessionID, input, formID); err != nil {
		return Operation{}, false, err
	}
	op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID))
	if err != nil {
		return Operation{}, false, err
	}
	return op, true, tx.Commit(ctx)
}

func (s Store) StartInterrupt(ctx context.Context, operationID, sessionID string) (Operation, bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Operation{}, false, err
	}
	defer tx.Rollback(ctx)
	if op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID)); err == nil {
		if op.Kind != "interrupt" || op.SessionID != sessionID {
			return Operation{}, false, ErrConflict
		}
		return op, false, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, err
	}
	var projectID, sessionState string
	err = tx.QueryRow(ctx, `SELECT project_id,state FROM opencode_sessions WHERE id=$1 FOR UPDATE`, sessionID).
		Scan(&projectID, &sessionState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, ErrSessionNotFound
	}
	if err != nil {
		return Operation{}, false, err
	}
	if sessionState != "ready" {
		return Operation{}, false, ErrSessionUnavailable
	}
	var interruptPending bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM opencode_operations
		WHERE session_id=$1 AND kind='interrupt' AND state IN ('submitting','accepted','running','uncertain'))`, sessionID).Scan(&interruptPending); err != nil {
		return Operation{}, false, err
	}
	if interruptPending {
		return Operation{}, false, ErrSessionUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO opencode_operations(id,project_id,session_id,kind,input,state)
		VALUES($1,$2,$3,'interrupt','interrupt','submitting')`, operationID, projectID, sessionID); err != nil {
		return Operation{}, false, err
	}
	op, err := scanOperation(tx.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID))
	if err != nil {
		return Operation{}, false, err
	}
	return op, true, tx.Commit(ctx)
}

func (s Store) MarkUncertain(ctx context.Context, operationID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE opencode_operations SET state='uncertain',error_code='opencode_result_uncertain',updated_at=now()
		WHERE id=$1 AND state='submitting'`, operationID)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `UPDATE opencode_sessions SET state='uncertain',updated_at=now()
		WHERE id=$1 AND state='creating' AND EXISTS (
			SELECT 1 FROM opencode_operations WHERE id=$1 AND kind='create_session'
		)`, operationID)
	return err
}

func (s Store) AcceptSession(ctx context.Context, operationID, externalID, title string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE opencode_sessions SET external_id=$2,title=$3,state='ready',updated_at=now()
		WHERE id=$1 AND state IN ('creating','uncertain')`, operationID, externalID, title); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE opencode_operations SET state='completed',external_id=$2,error_code=NULL,updated_at=now()
		WHERE id=$1 AND kind='create_session' AND state IN ('submitting','uncertain')`, operationID, externalID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s Store) AcceptPrompt(ctx context.Context, operationID, inboxID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE opencode_operations SET state='accepted',external_id=$2,error_code=NULL,updated_at=now()
		WHERE id=$1 AND kind='prompt' AND state='submitting'`, operationID, inboxID)
	return err
}

func (s Store) AcceptPermission(ctx context.Context, operationID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE opencode_operations SET state='completed',error_code=NULL,updated_at=now()
		WHERE id=$1 AND kind='permission' AND state='submitting'`, operationID)
	return err
}

func (s Store) AcceptForm(ctx context.Context, operationID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE opencode_operations SET state='completed',error_code=NULL,updated_at=now()
		WHERE id=$1 AND kind='form' AND state='submitting'`, operationID)
	return err
}

func (s Store) AcceptInterrupt(ctx context.Context, operationID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE opencode_operations SET state='accepted',error_code=NULL,updated_at=now()
		WHERE id=$1 AND kind='interrupt' AND state='submitting'`, operationID)
	return err
}

func (s Store) Operation(ctx context.Context, operationID string) (Operation, error) {
	return scanOperation(s.Pool.QueryRow(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE id=$1`, operationID))
}

func (s Store) Operations(ctx context.Context, sessionID string) ([]Operation, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,project_id,session_id,kind,input,state,external_id,error_code
		FROM opencode_operations WHERE session_id=$1 ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Operation, 0)
	for rows.Next() {
		item, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s Store) ReconcileOperation(ctx context.Context, operationID, state string, externalID *string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE opencode_operations SET state=$2,external_id=COALESCE($3,external_id),
		updated_at=now() WHERE id=$1 AND state IN ('accepted','running','uncertain')`, operationID, state, externalID)
	return err
}

func (s Store) Session(ctx context.Context, sessionID string) (SessionLink, error) {
	var result SessionLink
	err := s.Pool.QueryRow(ctx, `SELECT id,project_id,external_id,title,state,(extract(epoch FROM updated_at)*1000)::bigint
		FROM opencode_sessions WHERE id=$1`, sessionID).
		Scan(&result.ID, &result.ProjectID, &result.ExternalID, &result.Title, &result.State, &result.UpdatedAt)
	return result, err
}

func (s Store) Sessions(ctx context.Context, projectID string) ([]SessionLink, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,project_id,external_id,title,state,(extract(epoch FROM updated_at)*1000)::bigint
		FROM opencode_sessions WHERE project_id=$1 ORDER BY updated_at DESC LIMIT 100`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SessionLink, 0)
	for rows.Next() {
		var item SessionLink
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ExternalID, &item.Title, &item.State, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanOperation(row pgx.Row) (Operation, error) {
	var result Operation
	err := row.Scan(&result.ID, &result.ProjectID, &result.SessionID, &result.Kind, &result.Input,
		&result.State, &result.ExternalID, &result.ErrorCode)
	return result, err
}
