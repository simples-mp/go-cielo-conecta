package go_cielo_conecta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const maxLogSize = 1024 * 256 // 256 KB

type LogInfo struct {
	URL        string `json:"url"`
	Method     string `json:"method"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
}

func (l LogInfo) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("status", l.Status),
		slog.Int("status_code", l.StatusCode),
		slog.String("method", l.Method),
		slog.String("url", l.URL),
	)
}

func (c *Client) logHTTPRequest(r *http.Request) {
	if c.log == nil {
		return
	}

	c.LogInfo(
		"sending http request",
		"method", r.Method,
		"url", r.URL.String(),
		"headers", redactedHeaders(r.Header),
		"request_body", requestBodyLogValue(r),
	)
}

func (c *Client) logger(r *http.Request, resp *http.Response, responseBody []byte) {
	if c.log == nil {
		return
	}

	l := LogInfo{
		URL:        r.URL.String(),
		Method:     r.Method,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}

	if l.StatusCode < 200 || l.StatusCode > 299 {
		c.LogError("error executing the request", "info", l, "response_body", bodyLogValue(responseBody))
		return
	}

	c.LogInfo("request was successful", "info", l, "response_body", bodyLogValue(responseBody))
}

func redactedHeaders(headers http.Header) http.Header {
	clone := headers.Clone()
	for _, key := range []string{"Authorization", "Proxy-Authorization"} {
		if clone.Get(key) != "" {
			clone.Set(key, "<redacted>")
		}
	}

	return clone
}

func requestBodyLogValue(r *http.Request) any {
	if r.Body == nil {
		return nil
	}

	if r.GetBody != nil {
		body, err := r.GetBody()
		if err != nil {
			return fmt.Sprintf("failed to read request body: %v", err)
		}
		defer body.Close()

		data, err := io.ReadAll(body)
		if err != nil {
			return fmt.Sprintf("failed to read request body: %v", err)
		}

		return bodyLogValue(data)
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Sprintf("failed to read request body: %v", err)
	}

	r.Body = io.NopCloser(bytes.NewReader(data))
	return bodyLogValue(data)
}

func bodyLogValue(data []byte) any {
	if len(data) == 0 {
		return nil
	}

	if len(data) > maxLogSize {
		return map[string]any{
			"omitted": true,
			"bytes":   len(data),
			"limit":   maxLogSize,
		}
	}

	if json.Valid(data) {
		return json.RawMessage(data)
	}

	return string(data)
}

func (c *Client) SetLogger(logger *slog.Logger) {
	if logger == nil {
		c.log = nil
		return
	}

	c.log = logger.With("source", "cielo-conecta-client")
}

func (c *Client) DefaultLogger() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format(time.RFC3339))
			}
			return a
		},
	}))

	c.log = l.With("source", "cielo-conecta-client")
}

func (c *Client) LogInfo(msg string, args ...any) {
	if c.log == nil {
		return
	}

	c.log.Info(msg, args...)
}

func (c *Client) LogError(msg string, args ...any) {
	if c.log == nil {
		return
	}

	c.log.Error(msg, args...)
}
