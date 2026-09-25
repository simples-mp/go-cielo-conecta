package go_cielo_conecta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultUserAgent    = "go-cielo-conecta-client/1.0"
	maxResponseBodySize = 10 << 20 // 10 MiB
)

type Client struct {
	Client *http.Client

	env   Environment
	token *tokenResponse
	log   *slog.Logger

	tokenMu     sync.Mutex
	logPayloads atomic.Bool
}

type tokenResponse struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresIn   time.Duration `json:"expires_in"`

	expireTime time.Time `json:"-"`
}

type ClientInterface interface {
	NewRequest(method, path string, body any) (*http.Request, error)
	NewRequestWithContext(ctx context.Context, method, path string, body any) (*http.Request, error)
	Send(req *http.Request, body any) error

	CreateSale(info SaleInfo) SaleInterface

	GetPaymentByID(ctx context.Context, paymentId string) (Sale, error)
	GetPaymentByOrderID(ctx context.Context, orderID string, date ...time.Time) (Sale, error)
	ReversePayment(ctx context.Context, sale Sale) (ConfirmResponse, error)
	CancelPayment(ctx context.Context, sale Sale, merchantVoidId string) (ConfirmResponse, error)

	SharedLibrary(terminalID string, subMerchantId ...string) (map[string]any, error)

	SetLogger(slog *slog.Logger)
}

// NewClient creates a client and retrieves its initial access token.
// If token retrieval succeeds, it returns the initialized client.
func NewClient(env Environment, log ...*slog.Logger) (ClientInterface, error) {
	if env.merchant.ID == "" || env.merchant.Secret == "" || env.APIUrl == "" || env.OAuthURL == "" || env.APIQueryUrl == "" || env.ParamsURL == "" {
		return nil, errors.New("merchantId, merchantSecret and environment fields are required")
	}

	c := Client{
		Client: &http.Client{},
		env:    env,
		token:  nil,
	}

	c.DefaultLogger()

	if len(log) > 0 {
		c.SetLogger(log[0])
	}

	token, err := c.getToken(context.Background())
	if err != nil {
		return nil, err
	}

	c.LogInfo("cielo access token retrieved successfully", "expires_in", (token.ExpiresIn * time.Second).String())

	return &c, nil
}

// NewRequest creates a new HTTP requestBody with the specified method, path, and requestBody.
// If the requestBody is not nil, it encodes it as JSON and includes it in the requestBody.
//
// The function returns the created HTTP requestBody or an error if there was an issue encoding the requestBody.
func (c *Client) NewRequest(method, path string, body any) (*http.Request, error) {
	return c.NewRequestWithContext(context.Background(), method, path, body)
}

// NewRequestWithContext creates a new HTTP requestBody with the specified context, method, path, and requestBody.
// If the requestBody is not nil, it encodes it as JSON and includes it in the requestBody.
//
// The function returns the created HTTP requestBody or an error if there was an issue encoding the requestBody.
func (c *Client) NewRequestWithContext(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}

		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, path, buf)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// Send sends an HTTP requestBody and decodes the response into the provided variable.
// It sets the necessary headers for authentication and content type, and logs the requestBody and response.
//
// If the response status code indicates an error (not in the 200-299 range), it attempts to decode the error response
// and returns it. If there is an issue decoding the response, it returns an error with the status code and decoding error.
// If the requestBody is successful, it decodes the response requestBody into the provided variable.
func (c *Client) Send(req *http.Request, v any) error {
	if req == nil {
		return errors.New("send HTTP request: request is nil")
	}
	if c.Client == nil {
		return errors.New("send HTTP request: HTTP client is nil")
	}

	if c.env.Homologation {
		req.Header.Set("Environment", "Homologacao15")
	}

	token, err := c.getToken(req.Context())
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	c.logHTTPRequest(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	data, err := readResponseBody(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	c.logger(req, resp, data)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errCielo := MultiErr{}
		if err := json.Unmarshal(data, &errCielo); err == nil && len(errCielo) > 0 {
			return errCielo
		}

		return fmt.Errorf("HTTP request failed: status=%d", resp.StatusCode)
	}

	if v == nil || resp.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode response body (status=%d): %w", resp.StatusCode, err)
	}

	return nil
}

func readResponseBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBodySize {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseBodySize)
	}

	return data, nil
}

func (c *Client) getToken(ctx context.Context) (*tokenResponse, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != nil && time.Now().Before(c.token.expireTime) {
		return c.token, nil
	}

	body := bytes.NewBufferString("grant_type=client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.env.OAuthURL, body)
	if err != nil {
		return nil, fmt.Errorf("create OAuth request: %w", err)
	}

	if c.env.Homologation {
		req.Header.Set("Environment", "Homologacao15")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.env.merchant.ID, c.env.merchant.Secret)

	if c.Client == nil {
		return nil, errors.New("send OAuth request: HTTP client is nil")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send OAuth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := readResponseBody(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read OAuth error response (status=%d): %w", resp.StatusCode, err)
		}

		errCielo := MultiErr{}
		if err := json.Unmarshal(data, &errCielo); err == nil && len(errCielo) > 0 {
			return nil, errCielo
		}

		return nil, fmt.Errorf("OAuth request failed: status=%d", resp.StatusCode)
	}

	data, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read OAuth response: %w", err)
	}

	var token tokenResponse
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("decode OAuth response: %w", err)
	}

	const refreshMargin = 2 * time.Minute
	lifetime := token.ExpiresIn * time.Second
	margin := refreshMargin
	if lifetime <= margin {
		margin = lifetime / 10
	}
	token.expireTime = time.Now().Add(lifetime - margin)
	c.token = &token

	return c.token, nil
}
