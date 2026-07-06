package go_cielo_conecta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (c *Client) getToken() error {
	var body = bytes.NewBufferString("grant_type=client_credentials")

	req, err := http.NewRequest("POST", c.env.OAuthURL, body)
	if err != nil {
		return fmt.Errorf("cielo.getToken: %v", err)
	}

	if c.env.Homologation {
		req.Header.Set("Environment", "Homologacao15")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.env.merchant.ID, c.env.merchant.Secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("cielo.getToken: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cielo.getToken: request failed with status code %3d, failed to decode body=%v", resp.StatusCode, err)
		}

		return fmt.Errorf("cielo.getToken: request failed with status code %3d, body=%s", resp.StatusCode, string(data))
	}

	var token tokenResponse

	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return fmt.Errorf("cielo.getToken: %v", err)
	}

	c.token = &token

	return nil
}

func (c *Client) refreshToken(ctx context.Context) {
	waitDuration := (c.token.ExpiresIn * time.Second) - (5 * time.Minute)
	if waitDuration <= 0 {
		waitDuration = 10 * time.Second
	}

	ticker := time.NewTicker(waitDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.handleTokenRefresh(ticker)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) handleTokenRefresh(t *time.Ticker) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	err := c.getToken()
	if err != nil {
		t.Reset(time.Minute) // Try again in 1 minute if it was failed
		return
	}

	// Calculate the next interval with a safety margin
	nextIn := (c.token.ExpiresIn * time.Second) - (5 * time.Minute)
	if nextIn <= 0 {
		nextIn = 10 * time.Second
	}

	c.LogInfo(fmt.Sprintf("Token refreshed successfully, next refresh in %s\n", nextIn.String()))

	t.Reset(nextIn)
}

// Close cancels the token refresh goroutine and waits for it to finish.
func (c *Client) Close() {
	c.once.Do(func() {
		c.cancel()
		c.wg.Wait()
	})
}
