package eyeduxsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// successEnvelope wraps the "data" field of successful API responses.
type successEnvelope[T any] struct {
	Data T `json:"data"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// do executes an HTTP request and decodes the response into out.
// path must start with a leading slash (e.g. "/public/logs").
// body is JSON-serialized when non-nil; out receives the raw JSON response on success.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("eyedux: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("eyedux: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyedux: execute request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("eyedux: read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp, raw)
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("eyedux: decode response: %w", err)
		}
	}

	return nil
}

func parseAPIError(resp *http.Response, raw []byte) error {
	apiErr := &APIError{StatusCode: resp.StatusCode}

	var env errorEnvelope
	if json.Unmarshal(raw, &env) == nil {
		apiErr.Code = env.Error.Code
		apiErr.Message = env.Error.Message
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if v := resp.Header.Get("Retry-After"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				apiErr.RetryAfter = &n
			}
		}
	}

	return apiErr
}
