package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ Pool *pgxpool.Pool }
type Event struct {
	Version int    `json:"version"`
	Seq     int64  `json:"seq"`
	TurnID  string `json:"turn_id"`
	Type    string `json:"type"`
	Data    any    `json:"data"`
}

func (s Store) Add(ctx context.Context, turn, kind string, data any) (Event, error) {
	b, e := json.Marshal(data)
	if e != nil {
		return Event{}, e
	}
	v := Event{Version: 1, TurnID: turn, Type: kind, Data: data}
	e = s.Pool.QueryRow(ctx, "INSERT INTO conversation_events(turn_id,type,data) VALUES($1,$2,$3) RETURNING seq", turn, kind, b).Scan(&v.Seq)
	return v, e
}
func (s Store) Events(ctx context.Context, turn string, after int64) ([]Event, error) {
	rows, e := s.Pool.Query(ctx, "SELECT seq,type,data FROM conversation_events WHERE turn_id=$1 AND seq>$2 ORDER BY seq LIMIT 1000", turn, after)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		v := Event{Version: 1, TurnID: turn}
		var b []byte
		if e := rows.Scan(&v.Seq, &v.Type, &b); e != nil {
			return nil, e
		}
		if e := json.Unmarshal(b, &v.Data); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s Store) History(ctx context.Context, id string) ([]Message, error) {
	rows, e := s.Pool.Query(ctx, `SELECT input,generated,status FROM (SELECT input,generated,status,created_at FROM conversation_turns WHERE conversation_id=$1 AND status<>'running' ORDER BY created_at DESC LIMIT 12) t ORDER BY created_at`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Message{{Role: "system", Content: "Você é o Jarvis, assistente pessoal. Responda em português brasileiro, de forma breve e natural para fala. Use ferramentas somente quando necessário. Uma resposta interrompida pode não ter sido ouvida. Não alegue ter executado ações sem resultado de ferramenta."}}
	for rows.Next() {
		var a, b, c string
		if e := rows.Scan(&a, &b, &c); e != nil {
			return nil, e
		}
		out = append(out, Message{Role: "user", Content: a})
		if b != "" {
			if c != "completed" {
				b = "[Resposta gerada mas interrompida; reprodução não garantida] " + b
			}
			out = append(out, Message{Role: "assistant", Content: b})
		}
	}
	return out, rows.Err()
}

// A claimed execution is never automatically repeated, including after a crash.
func (s Store) Tool(ctx context.Context, turn string, c Call, execute func(string, string) (string, error)) (string, error) {
	var args struct {
		Input string `json:"input"`
	}
	if e := json.Unmarshal([]byte(c.Function.Arguments), &args); e != nil {
		return "", errors.New("invalid_tool_arguments")
	}
	if c.Function.Name != "time" && c.Function.Name != "calculator" {
		return "", errors.New("tool_requires_authorization")
	}
	tag, e := s.Pool.Exec(ctx, "INSERT INTO conversation_tools(turn_id,call_id,name,input,status) VALUES($1,$2,$3,$4,'claimed') ON CONFLICT DO NOTHING", turn, c.ID, c.Function.Name, args.Input)
	if e != nil {
		return "", e
	}
	if tag.RowsAffected() == 0 {
		var status, result, name, input string
		e = s.Pool.QueryRow(ctx, "SELECT status,result,name,input FROM conversation_tools WHERE turn_id=$1 AND call_id=$2", turn, c.ID).Scan(&status, &result, &name, &input)
		if e != nil {
			return "", e
		}
		if name != c.Function.Name || input != args.Input {
			return "", errors.New("tool_id_conflict")
		}
		if status == "completed" {
			return result, nil
		}
		return "", errors.New("tool_execution_uncertain")
	}
	result, runErr := execute(c.Function.Name, args.Input)
	status := "completed"
	if runErr != nil {
		status = "failed"
		result = runErr.Error()
	}
	_, e = s.Pool.Exec(context.WithoutCancel(ctx), "UPDATE conversation_tools SET status=$3,result=$4 WHERE turn_id=$1 AND call_id=$2", turn, c.ID, status, result)
	if e != nil {
		return "", e
	}
	return result, runErr
}
