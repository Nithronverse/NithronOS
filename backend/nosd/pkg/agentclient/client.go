package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

type Client struct {
	HTTP *http.Client
}

func New(socketPath string) *Client {
	return &Client{
		HTTP: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

func (c *Client) PostJSON(ctx context.Context, path string, body any, v any) error {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+path, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return &HTTPError{Status: res.StatusCode, Body: string(b)}
	}
	if v != nil {
		return json.NewDecoder(res.Body).Decode(v)
	}
	return nil
}

// GetJSON performs a GET and decodes JSON into v.
func (c *Client) GetJSON(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return &HTTPError{Status: res.StatusCode, Body: string(b)}
	}
	if v != nil {
		return json.NewDecoder(res.Body).Decode(v)
	}
	return nil
}

// HTTPError captures agent non-2xx responses
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("agent http %d: %s", e.Status, e.Body) }
