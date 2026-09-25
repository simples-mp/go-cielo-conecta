package go_cielo_conecta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxLogSize = 1024 * 256 // 256 KB

var sensitiveJSONFields = map[string]struct{}{
	"accesstoken":          {},
	"authorizationcode":    {},
	"birthday":             {},
	"card":                 {},
	"cardtoken":            {},
	"cardvoid":             {},
	"creditcard":           {},
	"customer":             {},
	"debitcard":            {},
	"email":                {},
	"emvdata":              {},
	"encryptedcarddata":    {},
	"encryptedpinblock":    {},
	"identity":             {},
	"initializationvector": {},
	"pinblock":             {},
	"securitycode":         {},
	"trackonedata":         {},
	"tracktwodata":         {},
}

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
		"request_body", requestBodyLogValue(r, c.logPayloads.Load()),
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

	if l.StatusCode < 200 || l.StatusCode >= 300 {
		c.LogError("error executing the request", "info", l, "response_body", bodyLogValue(responseBody, c.logPayloads.Load()))
		return
	}

	c.LogInfo("request was successful", "info", l, "response_body", bodyLogValue(responseBody, c.logPayloads.Load()))
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

func requestBodyLogValue(r *http.Request, enabled bool) any {
	if r.Body == nil {
		return nil
	}
	if !enabled {
		return omittedBodyLogValue("payload logging is disabled", r.ContentLength)
	}
	if r.GetBody == nil {
		return omittedBodyLogValue("request body cannot be replayed", r.ContentLength)
	}

	body, err := r.GetBody()
	if err != nil {
		return fmt.Sprintf("failed to read request body: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(io.LimitReader(body, maxLogSize+1))
	if err != nil {
		return fmt.Sprintf("failed to read request body: %v", err)
	}

	return bodyLogValue(data, true)
}

func bodyLogValue(data []byte, enabled bool) any {
	if len(data) == 0 {
		return nil
	}
	if !enabled {
		return omittedBodyLogValue("payload logging is disabled", int64(len(data)))
	}
	if len(data) > maxLogSize {
		return omittedBodyLogValue("payload exceeds logging limit", int64(len(data)))
	}
	if !json.Valid(data) {
		return omittedBodyLogValue("payload is not JSON", int64(len(data)))
	}

	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return omittedBodyLogValue("payload could not be decoded", int64(len(data)))
	}

	return redactJSONValue(value)
}

func omittedBodyLogValue(reason string, size int64) map[string]any {
	value := map[string]any{
		"omitted": true,
		"reason":  reason,
	}
	if size >= 0 {
		value["bytes"] = size
	}

	return value
}

func redactJSONValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		for key, item := range value {
			if _, sensitive := sensitiveJSONFields[normalizeJSONField(key)]; sensitive {
				value[key] = "<redacted>"
				continue
			}
			value[key] = redactJSONValue(item)
		}
	case []any:
		for i, item := range value {
			value[i] = redactJSONValue(item)
		}
	}

	return value
}

func normalizeJSONField(field string) string {
	field = strings.ToLower(field)
	field = strings.ReplaceAll(field, "_", "")
	return strings.ReplaceAll(field, "-", "")
}

func (c *Client) SetLogger(logger *slog.Logger) {
	if logger == nil {
		c.log = nil
		return
	}

	c.log = logger.With("source", "cielo-conecta-client")
}

// SetPayloadLogging controls whether JSON request and response bodies are included
// in logs. Sensitive fields are redacted even when it is enabled.
func (c *Client) SetPayloadLogging(enabled bool) {
	c.logPayloads.Store(enabled)
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
