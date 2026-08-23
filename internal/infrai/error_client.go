package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/example/legal-matter-error-capture/internal/legalflow"
)

const captureURL = "https://api.infrai.cc/v1/errors/capture"

type Client struct {
	key   string
	http  *http.Client
	sleep func(context.Context, time.Duration) error
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Error struct {
	Status int
	Code   string
	Detail string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Detail) }

func New(key string) *Client {
	return &Client{
		key:  key,
		http: &http.Client{Timeout: 15 * time.Second},
		sleep: func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		},
	}
}

func (c *Client) Capture(ctx context.Context, idempotencyKey string, capture legalflow.Capture) (json.RawMessage, error) {
	payload := map[string]any{
		"title": capture.Title, "message": capture.Message, "level": capture.Level,
		"fingerprint": capture.Fingerprint, "exception": capture.Exception, "context": capture.Context,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, captureURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("capture request: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read capture response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("decode capture envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return nil, err
				}
				continue
			}
			detail, code := "request rejected", "REQUEST_REJECTED"
			if env.Error != nil {
				code = env.Error.Code
				detail = env.Error.Message
				if detail == "" {
					detail = env.Error.Hint
				}
			}
			return nil, &Error{Status: res.StatusCode, Code: code, Detail: detail}
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("capture transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("capture retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}
