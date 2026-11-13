package conversation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/services"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	store  Store
	tools  *services.ToolService
	mu     sync.Mutex
	active map[string]context.CancelFunc
}

func Register(r *gin.Engine, s Store, tools *services.ToolService) {
	h := &Handler{store: s, tools: tools, active: map[string]context.CancelFunc{}}
	r.POST("/api/v2/conversations", h.create)
	r.GET("/api/v2/conversations/:id", h.history)
	r.POST("/api/v2/conversations/:id/turns", h.start)
	r.GET("/api/v2/turns/:turn/events", h.events)
	r.POST("/api/v2/turns/:turn/cancel", h.cancel)
	r.POST("/api/v2/turns/:turn/playback", h.playback)
}
func ID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func validID(s string) bool {
	if len(s) != 32 {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil
}
func (h *Handler) create(c *gin.Context) {
	id := ID()
	if _, e := h.store.Pool.Exec(c, "INSERT INTO conversations(id) VALUES($1)", id); e != nil {
		c.JSON(500, gin.H{"error": "storage_unavailable"})
		return
	}
	c.JSON(201, gin.H{"conversation_id": id})
}
func (h *Handler) history(c *gin.Context) {
	rows, e := h.store.Pool.Query(c, "SELECT id,input,generated,status,provider,model,played_segment,played_ms,delivery_status FROM conversation_turns WHERE conversation_id=$1 ORDER BY created_at DESC LIMIT 100", c.Param("id"))
	if e != nil {
		c.JSON(500, gin.H{"error": "storage_unavailable"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, input, text, status, provider, model, delivery string
		var segment, ms int
		if e := rows.Scan(&id, &input, &text, &status, &provider, &model, &segment, &ms, &delivery); e != nil {
			c.Status(500)
			return
		}
		out = append(out, gin.H{"turn_id": id, "input": input, "generated": text, "status": status, "provider": provider, "model": model, "played_segment": segment, "played_ms": ms, "delivery_status": delivery})
	}
	if rows.Err() != nil {
		c.Status(500)
		return
	}
	c.JSON(200, gin.H{"turns": out})
}
func (h *Handler) start(c *gin.Context) {
	var req struct {
		TurnID   string `json:"turn_id"`
		Message  string `json:"message"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32768)
	if c.ShouldBindJSON(&req) != nil || !validID(req.TurnID) || !validID(c.Param("id")) || strings.TrimSpace(req.Message) == "" || len(req.Message) > 8000 {
		c.JSON(400, gin.H{"error": "invalid_turn"})
		return
	}
	if req.Provider == "" {
		req.Provider = "scripted"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var existing, input, provider, model string
	e := h.store.Pool.QueryRow(c, "SELECT conversation_id,input,provider,model FROM conversation_turns WHERE id=$1", req.TurnID).Scan(&existing, &input, &provider, &model)
	if e == nil {
		if existing != c.Param("id") || input != req.Message || provider != req.Provider || model != req.Model {
			c.JSON(409, gin.H{"error": "turn_id_conflict"})
			return
		}
		c.JSON(200, gin.H{"turn_id": req.TurnID, "replayed": true})
		return
	}
	p, e := Resolve(req.Provider, req.Model)
	if e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	_, e = h.store.Pool.Exec(c, "INSERT INTO conversation_turns(id,conversation_id,input,provider,model) VALUES($1,$2,$3,$4,$5)", req.TurnID, c.Param("id"), req.Message, req.Provider, req.Model)
	if e != nil {
		c.JSON(409, gin.H{"error": "conversation_missing_or_turn_active"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	h.active[req.TurnID] = cancel
	go h.run(ctx, cancel, req.TurnID, c.Param("id"), req.Message, p)
	c.JSON(202, gin.H{"turn_id": req.TurnID})
}
func (h *Handler) run(ctx context.Context, cancel context.CancelFunc, turn, conversation, input string, p Provider) {
	defer cancel()
	defer func() { h.mu.Lock(); delete(h.active, turn); h.mu.Unlock() }()
	generated := ""
	status := "completed"
	emit := func(kind string, data any) error { _, e := h.store.Add(ctx, turn, kind, data); return e }
	messages, err := h.store.History(ctx, conversation)
	if err == nil {
		messages = append(messages, Message{Role: "user", Content: input})
		err = emit("turn.started", gin.H{"conversation_id": conversation})
	}
	for round := 0; err == nil; round++ {
		if round >= 4 {
			err = errors.New("tool_round_limit")
			break
		}
		var calls []Call
		roundText := ""
		calls, err = p.Stream(ctx, messages, func(s string) error {
			if len(generated)+len(s) > 64000 {
				return errors.New("response_limit")
			}
			generated += s
			roundText += s
			return emit("text.delta", gin.H{"delta": s})
		})
		if err != nil || len(calls) == 0 {
			break
		}
		messages = append(messages, Message{Role: "assistant", Content: roundText, ToolCalls: calls})
		for _, call := range calls {
			if err = emit("tool.call", call); err != nil {
				break
			}
			result, toolErr := h.store.Tool(ctx, turn, call, func(name, input string) (string, error) { return h.tools.ExecuteTool(ctx, name, input) })
			if toolErr != nil {
				result = toolErr.Error()
			}
			if err = emit("tool.result", gin.H{"call_id": call.ID, "result": result, "failed": toolErr != nil}); err != nil {
				break
			}
			messages = append(messages, Message{Role: "tool", Content: result, ToolCallID: call.ID})
		}
	}
	finish, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	if err != nil {
		status = "failed"
		code := "provider_unavailable"
		var f *Failure
		if errors.As(err, &f) {
			code = f.Code
		}
		if ctx.Err() != nil {
			status = "interrupted"
			code = "cancelled"
		}
		_, _ = h.store.Add(finish, turn, "error", gin.H{"code": code, "cancel_state": "local_discard", "detail": safeDetail(err), "retry_after": func() string {
			if f != nil {
				return f.RetryAfter
			}
			return ""
		}()})
	}
	_, saveErr := h.store.Pool.Exec(finish, "UPDATE conversation_turns SET generated=$2,status=$3,finished_at=now() WHERE id=$1", turn, generated, status)
	if saveErr == nil {
		_, _ = h.store.Add(finish, turn, "turn.finished", gin.H{"status": status})
	}
}
func safeDetail(e error) string {
	var f *Failure
	if errors.As(e, &f) {
		return f.Code
	}
	if errors.Is(e, context.Canceled) {
		return "cancelled"
	}
	return "request_failed"
}
func (h *Handler) cancel(c *gin.Context) {
	h.mu.Lock()
	cancel := h.active[c.Param("turn")]
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if _, e := h.store.Pool.Exec(c, "UPDATE conversation_turns SET delivery_status='interrupted' WHERE id=$1", c.Param("turn")); e != nil {
		c.Status(500)
		return
	}
	c.JSON(200, gin.H{"cancel_state": "local_discard", "provider_confirmed": false})
}
func (h *Handler) events(c *gin.Context) {
	var exists bool
	if e := h.store.Pool.QueryRow(c, "SELECT EXISTS(SELECT 1 FROM conversation_turns WHERE id=$1)", c.Param("turn")).Scan(&exists); e != nil || !exists {
		c.JSON(404, gin.H{"error": "turn_not_found"})
		return
	}
	after, _ := strconv.ParseInt(c.Query("after"), 10, 64)
	if s := c.GetHeader("Last-Event-ID"); s != "" {
		after, _ = strconv.ParseInt(s, 10, 64)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	timer := time.NewTicker(80 * time.Millisecond)
	defer timer.Stop()
	for {
		events, e := h.store.Events(c.Request.Context(), c.Param("turn"), after)
		if e != nil {
			return
		}
		for _, v := range events {
			b, _ := json.Marshal(v)
			if _, e = fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", v.Seq, v.Type, b); e != nil {
				return
			}
			after = v.Seq
			c.Writer.Flush()
			if v.Type == "turn.finished" {
				return
			}
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-timer.C:
		}
	}
}
func (h *Handler) playback(c *gin.Context) {
	var v struct {
		Segment  int  `json:"segment"`
		MS       int  `json:"ms"`
		Finished bool `json:"finished"`
	}
	if c.ShouldBindJSON(&v) != nil || v.Segment < 0 || v.MS < 0 || v.MS > 3600000 || v.Segment > 10000 {
		c.Status(400)
		return
	}
	_, e := h.store.Pool.Exec(c, "UPDATE conversation_turns SET played_segment=$2,played_ms=$3 WHERE id=$1 AND (played_segment<$2 OR (played_segment=$2 AND played_ms<$3))", c.Param("turn"), v.Segment, v.MS)
	if e != nil {
		c.Status(500)
		return
	}
	if v.Finished {
		if _, e = h.store.Pool.Exec(c, "UPDATE conversation_turns SET delivery_status='finished_estimated' WHERE id=$1 AND delivery_status<>'interrupted'", c.Param("turn")); e != nil {
			c.Status(500)
			return
		}
	}
	c.Status(204)
}
