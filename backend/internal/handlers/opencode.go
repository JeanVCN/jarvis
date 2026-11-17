package handlers

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/opencode"
)

func RegisterOpenCode(router *gin.Engine, client *opencode.Client, store opencode.Store, writeEnabled bool) {
	router.GET("/api/opencode/status", func(c *gin.Context) {
		result, err := client.Status(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
			return
		}
		result.WriteEnabled = result.WriteEnabled && writeEnabled
		c.JSON(http.StatusOK, result)
	})
	router.GET("/api/opencode/projects", func(c *gin.Context) {
		result, err := client.Projects(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": result})
	})
	router.GET("/api/opencode/projects/:project/sessions", func(c *gin.Context) {
		result, err := client.Sessions(c.Request.Context(), c.Param("project"))
		if err != nil {
			if err.Error() == "invalid_project_id" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_project_id"})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessions": result})
	})
	router.GET("/api/opencode/projects/:project/jarvis-sessions", func(c *gin.Context) {
		result, err := store.Sessions(c.Request.Context(), c.Param("project"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessions": result})
	})
	router.POST("/api/opencode/projects/:project/sessions", func(c *gin.Context) {
		if !writeEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "opencode_write_disabled"})
			return
		}
		var request struct {
			OperationID string `json:"operation_id"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		if c.ShouldBindJSON(&request) != nil || !validOperationID(request.OperationID) || !validProjectID(c.Param("project")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		op, created, err := store.StartSession(c.Request.Context(), request.OperationID, c.Param("project"))
		if err != nil {
			if errors.Is(err, opencode.ErrConflict) {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable"})
			return
		}
		if !created {
			c.JSON(http.StatusOK, gin.H{"operation": op})
			return
		}

		callCtx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		createdSession, err := client.CreateSession(callCtx, op.ProjectID, op.ID)
		cancel()
		if err != nil {
			markErr := store.MarkUncertain(context.Background(), op.ID)
			if markErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable", "operation_id": op.ID})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"operation": gin.H{"id": op.ID, "project_id": op.ProjectID, "session_id": op.SessionID, "kind": op.Kind, "state": "uncertain"}})
			return
		}
		persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = store.AcceptSession(persistCtx, op.ID, createdSession.ID, createdSession.Title)
		persistCancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable", "operation_id": op.ID})
			return
		}
		op.State = "completed"
		op.ExternalID = &createdSession.ID
		c.JSON(http.StatusCreated, gin.H{"operation": op})
	})
	router.POST("/api/opencode/sessions/:session/prompts", func(c *gin.Context) {
		if !writeEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "opencode_write_disabled"})
			return
		}
		var request struct {
			OperationID string `json:"operation_id"`
			Text        string `json:"text"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 12*1024)
		if c.ShouldBindJSON(&request) != nil || !validOperationID(request.OperationID) || !validOperationID(c.Param("session")) || strings.TrimSpace(request.Text) == "" || len(request.Text) > 8000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_prompt"})
			return
		}
		op, created, err := store.StartPrompt(c.Request.Context(), request.OperationID, c.Param("session"), request.Text)
		if err != nil {
			switch {
			case errors.Is(err, opencode.ErrConflict):
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			case errors.Is(err, opencode.ErrSessionNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			case errors.Is(err, opencode.ErrSessionUnavailable):
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable"})
			}
			return
		}
		if !created {
			c.JSON(http.StatusOK, gin.H{"operation": op})
			return
		}
		link, err := store.Session(c.Request.Context(), op.SessionID)
		if err != nil || link.ExternalID == nil {
			_ = store.MarkUncertain(context.Background(), op.ID)
			c.JSON(http.StatusAccepted, gin.H{"operation": gin.H{"id": op.ID, "session_id": op.SessionID, "state": "uncertain"}})
			return
		}
		callCtx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		accepted, err := client.Prompt(callCtx, op.ProjectID, *link.ExternalID, op.ID, op.Input)
		cancel()
		if err != nil {
			markErr := store.MarkUncertain(context.Background(), op.ID)
			if markErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable", "operation_id": op.ID})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"operation": gin.H{"id": op.ID, "project_id": op.ProjectID, "session_id": op.SessionID, "kind": op.Kind, "state": "uncertain"}})
			return
		}
		persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = store.AcceptPrompt(persistCtx, op.ID, accepted.InboxID)
		persistCancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable", "operation_id": op.ID})
			return
		}
		op.State = "accepted"
		op.ExternalID = &accepted.InboxID
		c.JSON(http.StatusAccepted, gin.H{"operation": op})
	})
	router.POST("/api/opencode/sessions/:session/permissions/:request/reply", func(c *gin.Context) {
		if !writeEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "opencode_write_disabled"})
			return
		}
		var request struct {
			OperationID string `json:"operation_id"`
			Decision    string `json:"decision"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		if c.ShouldBindJSON(&request) != nil || !validOperationID(request.OperationID) || (request.Decision != "once" && request.Decision != "reject") || len(c.Param("request")) > 128 || c.Param("request") == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_permission_decision"})
			return
		}
		op, created, err := store.StartPermission(c.Request.Context(), request.OperationID, c.Param("session"), c.Param("request"), request.Decision)
		if err != nil {
			switch {
			case errors.Is(err, opencode.ErrConflict):
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			case errors.Is(err, opencode.ErrSessionNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			case errors.Is(err, opencode.ErrSessionUnavailable):
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable"})
			}
			return
		}
		if !created {
			c.JSON(http.StatusOK, gin.H{"operation": op})
			return
		}
		link, err := store.Session(c.Request.Context(), op.SessionID)
		if err != nil || link.ExternalID == nil {
			_ = store.MarkUncertain(context.Background(), op.ID)
			c.JSON(http.StatusAccepted, gin.H{"operation": gin.H{"id": op.ID, "session_id": op.SessionID, "state": "uncertain"}})
			return
		}
		decision := request.Decision
		requestID := c.Param("request")
		callCtx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		err = client.ReplyPermission(callCtx, op.ProjectID, *link.ExternalID, requestID, op.ID, decision)
		cancel()
		if err != nil {
			markErr := store.MarkUncertain(context.Background(), op.ID)
			if markErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable", "operation_id": op.ID})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"operation": gin.H{"id": op.ID, "project_id": op.ProjectID, "session_id": op.SessionID, "kind": op.Kind, "state": "uncertain"}})
			return
		}
		persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = store.AcceptPermission(persistCtx, op.ID)
		persistCancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable", "operation_id": op.ID})
			return
		}
		op.State = "completed"
		c.JSON(http.StatusOK, gin.H{"operation": op})
	})
	router.GET("/api/opencode/sessions/:session", func(c *gin.Context) {
		if !validOperationID(c.Param("session")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
			return
		}
		link, err := store.Session(c.Request.Context(), c.Param("session"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "opencode_session_not_found"})
			return
		}
		if link.State == "uncertain" && link.ExternalID == nil {
			lookupCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
			created, lookupErr := client.FindCreatedSession(lookupCtx, link.ProjectID, link.ID)
			cancel()
			if lookupErr == nil && created != nil {
				persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if store.AcceptSession(persistCtx, link.ID, created.ID, created.Title) == nil {
					link, _ = store.Session(c.Request.Context(), link.ID)
				}
				persistCancel()
			}
		}
		var detail *opencode.SessionDetail
		if link.ExternalID != nil && link.State == "ready" {
			value, err := client.Session(c.Request.Context(), link.ProjectID, *link.ExternalID)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
				return
			}
			detail = &value
			reconcileSessionOperations(c, store, link, value)
		}
		operations, err := store.Operations(c.Request.Context(), link.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"session": link, "detail": detail, "operations": operations})
	})
}

func reconcileSessionOperations(c *gin.Context, store opencode.Store, link opencode.SessionLink, detail opencode.SessionDetail) {
	operations, err := store.Operations(c.Request.Context(), link.ID)
	if err != nil {
		return
	}
	for _, operation := range operations {
		if operation.Kind != "prompt" || operation.State == "completed" || operation.State == "failed" {
			continue
		}
		inboxID := ""
		state := operation.State
		if operation.ExternalID != nil {
			inboxID = *operation.ExternalID
			if state == "uncertain" || state == "submitting" {
				state = "accepted"
			}
		} else {
			for _, message := range detail.Messages {
				if message.Role == "user" && message.ID == operation.ID && message.Content == operation.Input {
					inboxID = message.ID
					break
				}
			}
		}
		var externalID *string
		if inboxID != "" {
			externalID = &inboxID
			found := false
			for _, message := range detail.Messages {
				if message.Role == "user" && message.ID == inboxID {
					found = true
					continue
				}
				if found && message.Role == "assistant" {
					if message.Completed {
						state = "completed"
						if message.Finish == "error" || detail.Session.Outcome != nil && *detail.Session.Outcome == "failed" {
							state = "failed"
						}
					}
					break
				}
			}
			if state != "completed" && state != "failed" && detail.State == "running" {
				state = "running"
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = store.ReconcileOperation(ctx, operation.ID, state, externalID)
		cancel()
	}
}

func validOperationID(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func validProjectID(value string) bool {
	return projectIDPattern.MatchString(value)
}

var projectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
