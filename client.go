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
	"time"
)

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

	Close()
	SetLogger(slog *slog.Logger)
}

// NewClient creates a new Client struct, retrieves an access token, and starts a goroutine to refresh the token periodically.
// If the token retrieval is successful, it returns the initialized Client instance. Otherwise, it returns an error.
func NewClient(env Environment, log ...*slog.Logger) (ClientInterface, error) {
	if env.merchant.ID == "" || env.merchant.Secret == "" || env.APIUrl == "" || env.OAuthURL == "" || env.APIQueryUrl == "" || env.ParamsURL == "" {
		return nil, errors.New("merchantId, merchantSecret and environment fields are required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := Client{
		Mutex:  sync.Mutex{},
		Client: &http.Client{},
		env:    env,
		cancel: cancel,
		token:  nil,
	}

	c.DefaultLogger()

	if len(log) > 0 {
		c.SetLogger(log[0])
	}

	err := c.getToken()
	if err != nil {
		return nil, err
	}

	c.LogInfo(fmt.Sprintf("Cielo access_token expires in %s", (c.token.ExpiresIn * time.Second).String()))

	c.wg.Go(func() {
		c.refreshToken(ctx)
	})

	return &c, nil
}

// NewRequest creates a new HTTP requestBody with the specified method, path, and requestBody.
// If the requestBody is not nil, it encodes it as JSON and includes it in the requestBody.
//
// The function returns the created HTTP requestBody or an error if there was an issue encoding the requestBody.
func (c *Client) NewRequest(method, path string, body any) (*http.Request, error) {
	var buf bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	return http.NewRequest(method, path, &buf)
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
			return nil, err
		}

		buf = bytes.NewBuffer(b)
	}

	return http.NewRequestWithContext(ctx, method, path, buf)
}

// Send sends an HTTP requestBody and decodes the response into the provided variable.
// It sets the necessary headers for authentication and content type, and logs the requestBody and response.
//
// If the response status code indicates an error (not in the 200-299 range), it attempts to decode the error response
// and returns it. If there is an issue decoding the response, it returns an error with the status code and decoding error.
// If the requestBody is successful, it decodes the response requestBody into the provided variable.
func (c *Client) Send(req *http.Request, v any) error {
	if v == nil {
		return nil
	}

	if c.env.Homologation {
		req.Header.Set("Environment", "Homologacao15")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-cielo-conecta-client/1.0")
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	c.logHTTPRequest(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	c.logger(req, resp, data)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		errCielo := MultiErr{}

		err = json.Unmarshal(data, &errCielo)
		if err == nil && len(errCielo) > 0 {
			return errCielo
		}

		return fmt.Errorf("request failed, status=%d, body=%s", resp.StatusCode, string(data))
	}

	return json.NewDecoder(bytes.NewReader(data)).Decode(v)
}
