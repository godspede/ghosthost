// internal/cli/client.go
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
)

// Client talks to the local daemon's admin API.
type Client struct {
	BaseURL string
	Secret  string
	HTTP    *http.Client
}

func NewClient(port int, secret string) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		Secret:  secret,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Secret)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, string(b))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(b, out)
}

func (c *Client) Share(ctx context.Context, req admin.ShareRequest) (admin.SharePayload, error) {
	var p admin.SharePayload
	err := c.do(ctx, "POST", "/share", req, &p)
	return p, err
}

func (c *Client) Revoke(ctx context.Context, id string) error {
	return c.do(ctx, "POST", "/revoke", admin.IDRequest{ID: id}, nil)
}

func (c *Client) Reshare(ctx context.Context, id string) (admin.SharePayload, error) {
	var p admin.SharePayload
	err := c.do(ctx, "POST", "/reshare", admin.IDRequest{ID: id}, &p)
	return p, err
}

func (c *Client) List(ctx context.Context) (admin.ListResponse, error) {
	var r admin.ListResponse
	err := c.do(ctx, "GET", "/list", nil, &r)
	return r, err
}

func (c *Client) Status(ctx context.Context) (admin.StatusResponse, error) {
	var r admin.StatusResponse
	err := c.do(ctx, "GET", "/status", nil, &r)
	return r, err
}

func (c *Client) Stop(ctx context.Context) error {
	return c.do(ctx, "POST", "/stop", nil, nil)
}

// ErrUnreachable is returned when the daemon is not running.
var ErrUnreachable = errors.New("daemon unreachable")
