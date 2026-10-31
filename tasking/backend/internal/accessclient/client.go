package accessclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aether/constellation/access"
	"aether/constellation/tasks"
)

// ErrUnavailable means the constellation could not be reached.
var ErrUnavailable = errors.New("constellation catalog unavailable")

// BadRequest is a constellation rejection of the request body.
type BadRequest struct {
	Message string
}

func (e *BadRequest) Error() string { return e.Message }

// Client calls the constellation for a collection and for the hold on a satellite.
type Client struct {
	base   string
	client *http.Client
}

// New talks to the constellation base URL.
func New(baseURL string) *Client {
	return &Client{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Earliest asks which collection is free. The constellation reads its task table.
func (c *Client) Earliest(ctx context.Context, req access.Request) (access.Offer, bool, error) {
	body, status, err := c.post(ctx, "/api/acquisitions", req, "")
	if err != nil {
		return access.Offer{}, false, err
	}
	if status == http.StatusBadRequest {
		return access.Offer{}, false, badRequest(body)
	}
	if status != http.StatusOK {
		return access.Offer{}, false, ErrUnavailable
	}
	return access.UnmarshalResult(body)
}

// Create stores a task for the earliest free collection.
func (c *Client) Create(ctx context.Context, req access.Request, clientRef string) (tasks.Task, error) {
	body, status, err := c.post(ctx, "/api/tasks", req, clientRef)
	if err != nil {
		return tasks.Task{}, err
	}
	if status == http.StatusBadRequest {
		return tasks.Task{}, badRequest(body)
	}
	if status == http.StatusConflict {
		return tasks.Task{}, tasks.ErrNotFeasible
	}
	if status != http.StatusOK {
		return tasks.Task{}, ErrUnavailable
	}
	return tasks.Unmarshal(body)
}

// Cancel asks the constellation to cancel the task.
func (c *Client) Cancel(ctx context.Context, id string) (tasks.Task, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/tasks/"+url.PathEscape(id)+"/cancel", nil)
	if err != nil {
		return tasks.Task{}, err
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return tasks.Task{}, ErrUnavailable
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tasks.Task{}, ErrUnavailable
	}
	if resp.StatusCode == http.StatusConflict {
		return tasks.Task{}, tasks.ErrFinished
	}
	if resp.StatusCode == http.StatusNotFound {
		return tasks.Task{}, tasks.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return tasks.Task{}, ErrUnavailable
	}
	return tasks.Unmarshal(body)
}

// Query reads the current status of the given tasks.
func (c *Client) Query(ctx context.Context, ids []string) ([]tasks.Task, error) {
	payload, err := json.Marshal(map[string]any{"ids": ids})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/tasks/query", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrUnavailable
	}
	return tasks.UnmarshalAll(body)
}

func (c *Client) post(ctx context.Context, path string, req access.Request, clientRef string) ([]byte, int, error) {
	raw, err := access.Encode(req, clientRef)
	if err != nil {
		return nil, 0, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	return body, resp.StatusCode, nil
}

func badRequest(body []byte) error {
	var payload struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)
	if payload.Error == "" {
		payload.Error = "bad request"
	}
	return &BadRequest{Message: payload.Error}
}
