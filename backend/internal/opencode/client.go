package opencode

import (
	"context"
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
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
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
		http:    &http.Client{Timeout: 5 * time.Second},
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

func (c *Client) get(ctx context.Context, path string, target any) error {
	u := *c.baseURL
	u.Path += path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.New("opencode_bridge_request_invalid")
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
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
