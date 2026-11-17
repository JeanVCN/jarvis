package opencode

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var projectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type Status struct {
	Status       string `json:"status"`
	Version      string `json:"version,omitempty"`
	WriteEnabled bool   `json:"write_enabled,omitempty"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Session struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	UpdatedAt int64   `json:"updated_at"`
	Outcome   *string `json:"outcome"`
}

type SessionCreated struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	UpdatedAt int64   `json:"updated_at"`
	Outcome   *string `json:"outcome"`
}

type PromptAccepted struct {
	InboxID string `json:"inbox_id"`
}

type Message struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
	Completed bool   `json:"completed"`
	Finish    string `json:"finish,omitempty"`
}

type DiffFile struct {
	File      string `json:"file"`
	Patch     string `json:"patch"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"`
}

type Permission struct {
	ID        string   `json:"id"`
	Action    string   `json:"action"`
	Resources []string `json:"resources"`
	Message   string   `json:"message,omitempty"`
}

type FormOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Recommended bool   `json:"recommended,omitempty"`
}

type FormField struct {
	Key         string       `json:"key"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Type        string       `json:"type"`
	Custom      bool         `json:"custom,omitempty"`
	Options     []FormOption `json:"options,omitempty"`
}

type Form struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Fields []FormField `json:"fields"`
}

type SessionDetail struct {
	Session     Session      `json:"session"`
	State       string       `json:"state"`
	Messages    []Message    `json:"messages"`
	Diff        []DiffFile   `json:"diff"`
	Permissions []Permission `json:"permissions"`
	Forms       []Form       `json:"forms"`
}

type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) (*Client, error) {
	if len(token) < 32 {
		return nil, errors.New("opencode_bridge_token_invalid")
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil || parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("opencode_bridge_url_invalid")
	}
	host := parsed.Hostname()
	if host == "" || !isLoopback(host) {
		return nil, errors.New("opencode_bridge_must_be_loopback")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return &Client{
		baseURL: parsed,
		token:   token,
		http: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var result Status
	err := c.get(ctx, "/v1/status", &result)
	return result, err
}

func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	var result struct {
		Projects []Project `json:"projects"`
	}
	if err := c.get(ctx, "/v1/projects", &result); err != nil {
		return nil, err
	}
	if result.Projects == nil {
		return []Project{}, nil
	}
	return result.Projects, nil
}

func (c *Client) Sessions(ctx context.Context, projectID string) ([]Session, error) {
	if !projectIDPattern.MatchString(projectID) {
		return nil, errors.New("invalid_project_id")
	}
	var result struct {
		Sessions []Session `json:"sessions"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions"
	if err := c.get(ctx, path, &result); err != nil {
		return nil, err
	}
	if result.Sessions == nil {
		return []Session{}, nil
	}
	return result.Sessions, nil
}

func (c *Client) CreateSession(ctx context.Context, projectID, operationID string) (SessionCreated, error) {
	var result SessionCreated
	if !projectIDPattern.MatchString(projectID) || !validOperationID(operationID) {
		return result, errors.New("opencode_request_invalid")
	}
	body := map[string]string{"operation_id": operationID}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions"
	err := c.post(ctx, path, body, &result)
	return result, err
}

func (c *Client) FindCreatedSession(ctx context.Context, projectID, operationID string) (*SessionCreated, error) {
	if !projectIDPattern.MatchString(projectID) || !validOperationID(operationID) {
		return nil, errors.New("opencode_request_invalid")
	}
	var result struct {
		Session *SessionCreated `json:"session"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/operations/" + url.PathEscape(operationID)
	if err := c.get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Session, nil
}

func (c *Client) Prompt(ctx context.Context, projectID, sessionID, operationID, text string) (PromptAccepted, error) {
	var result PromptAccepted
	if !projectIDPattern.MatchString(projectID) || !validOperationID(operationID) || sessionID == "" {
		return result, errors.New("opencode_request_invalid")
	}
	body := map[string]string{"operation_id": operationID, "text": text}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions/" + url.PathEscape(sessionID) + "/prompts"
	err := c.post(ctx, path, body, &result)
	return result, err
}

func (c *Client) PromptImage(ctx context.Context, projectID, sessionID, operationID, text, mimeType string, image []byte) (PromptAccepted, error) {
	var result PromptAccepted
	if !projectIDPattern.MatchString(projectID) || !validOperationID(operationID) || sessionID == "" ||
		len(text) == 0 || len(text) > 8000 || len(image) == 0 || len(image) > 10<<20 || !supportedImageMIME(mimeType) {
		return result, errors.New("opencode_request_invalid")
	}
	body := map[string]any{
		"operation_id": operationID,
		"text":         text,
		"image": map[string]string{
			"mime":   mimeType,
			"base64": base64.StdEncoding.EncodeToString(image),
		},
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions/" + url.PathEscape(sessionID) + "/prompts"
	err := c.post(ctx, path, body, &result)
	return result, err
}

func (c *Client) ReplyPermission(ctx context.Context, projectID, sessionID, requestID, operationID, decision string) error {
	if !projectIDPattern.MatchString(projectID) || sessionID == "" || requestID == "" || !validOperationID(operationID) || (decision != "once" && decision != "reject") {
		return errors.New("opencode_request_invalid")
	}
	body := map[string]string{"operation_id": operationID, "decision": decision}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions/" + url.PathEscape(sessionID) + "/permissions/" + url.PathEscape(requestID) + "/reply"
	var result map[string]any
	return c.post(ctx, path, body, &result)
}

func (c *Client) ReplyForm(ctx context.Context, projectID, sessionID, formID, operationID string, answer map[string]any) error {
	if !projectIDPattern.MatchString(projectID) || sessionID == "" || formID == "" || !validOperationID(operationID) || len(answer) == 0 {
		return errors.New("opencode_request_invalid")
	}
	body := map[string]any{"operation_id": operationID, "answer": answer}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions/" + url.PathEscape(sessionID) + "/forms/" + url.PathEscape(formID) + "/reply"
	var result map[string]any
	return c.post(ctx, path, body, &result)
}

func (c *Client) Interrupt(ctx context.Context, projectID, sessionID, operationID string) error {
	if !projectIDPattern.MatchString(projectID) || sessionID == "" || !validOperationID(operationID) {
		return errors.New("opencode_request_invalid")
	}
	body := map[string]string{"operation_id": operationID}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions/" + url.PathEscape(sessionID) + "/interrupt"
	var result map[string]any
	return c.post(ctx, path, body, &result)
}

func (c *Client) Session(ctx context.Context, projectID, sessionID string) (SessionDetail, error) {
	var result SessionDetail
	if !projectIDPattern.MatchString(projectID) || sessionID == "" {
		return result, errors.New("opencode_request_invalid")
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/sessions/" + url.PathEscape(sessionID)
	err := c.get(ctx, path, &result)
	return result, err
}

func validOperationID(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func supportedImageMIME(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	return c.request(ctx, http.MethodGet, path, nil, target)
}

func (c *Client) post(ctx context.Context, path string, body any, target any) error {
	return c.request(ctx, http.MethodPost, path, body, target)
}

func (c *Client) request(ctx context.Context, method, path string, body, target any) error {
	u := *c.baseURL
	u.Path += path
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return errors.New("opencode_bridge_request_invalid")
		}
		requestBody = strings.NewReader(string(encoded))
	}
	request, err := http.NewRequestWithContext(ctx, method, u.String(), requestBody)
	if err != nil {
		return errors.New("opencode_bridge_request_invalid")
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return errors.New("opencode_bridge_unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("opencode_bridge_http_%d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return errors.New("opencode_bridge_response_invalid")
	}
	return nil
}
