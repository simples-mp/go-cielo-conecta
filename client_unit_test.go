package go_cielo_conecta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRequestUsesConsistentJSONEncoding(t *testing.T) {
	client := &Client{}
	body := map[string]string{"message": "hello"}

	req, err := client.NewRequest(http.MethodPost, "https://example.com", body)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(data), `{"message":"hello"}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if req.GetBody == nil {
		t.Fatal("GetBody is nil; request body should be replayable")
	}
}

func TestNewRequestWithContextWrapsJSONError(t *testing.T) {
	client := &Client{}
	_, err := client.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com", make(chan int))
	if err == nil || !strings.Contains(err.Error(), "encode request body") {
		t.Fatalf("error = %v, want contextual JSON encoding error", err)
	}
}

func TestSendExecutesRequestWhenDestinationIsNil(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := clientWithToken(server.Client())
	req, err := client.NewRequest(http.MethodPost, server.URL, map[string]string{"value": "sent"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Send(req, nil); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("request calls = %d, want 1", got)
	}
}

func TestSendAcceptsSuccessfulEmptyResponses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()

			client := clientWithToken(server.Client())
			req, err := client.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			var response map[string]any
			if err := client.Send(req, &response); err != nil {
				t.Fatalf("Send returned an error for an empty %d response: %v", status, err)
			}
		})
	}
}

func TestSendRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte("x"), maxResponseBodySize+1))
	}))
	defer server.Close()

	client := clientWithToken(server.Client())
	req, err := client.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	var response any
	err = client.Send(req, &response)
	if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
		t.Fatalf("error = %v, want response size error", err)
	}
}

func TestSendDoesNotIncludeErrorBody(t *testing.T) {
	const secret = "sensitive-card-data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, secret, http.StatusBadGateway)
	}))
	defer server.Close()

	client := clientWithToken(server.Client())
	req, err := client.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Send(req, nil)
	if err == nil {
		t.Fatal("Send returned nil, want an HTTP status error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked response body: %v", err)
	}
}

func TestSendPropagatesContextToTokenRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		select {
		case <-r.Context().Done():
		case <-releaseHandler:
		}
	}))
	defer server.Close()

	client := &Client{
		Client: server.Client(),
		env:    Environment{OAuthURL: server.URL},
	}
	ctx, cancel := context.WithCancel(context.Background())
	req, err := client.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api", nil)
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- client.Send(req, nil) }()
	<-requestStarted
	cancel()

	select {
	case err = <-errCh:
		close(releaseHandler)
	case <-time.After(time.Second):
		close(releaseHandler)
		<-errCh
		t.Fatal("Send did not cancel the OAuth request")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestConcurrentSendRefreshesTokenOnce(t *testing.T) {
	var tokenCalls atomic.Int32
	var apiCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls.Add(1)
			time.Sleep(20 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"fresh","expires_in":3600}`)
		case "/api":
			apiCalls.Add(1)
			if got := r.Header.Get("Authorization"); got != "Bearer fresh" {
				t.Errorf("Authorization = %q, want Bearer fresh", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{
		Client: server.Client(),
		env:    Environment{OAuthURL: server.URL + "/token"},
	}
	const goroutines = 12
	start := make(chan struct{})
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, err := client.NewRequest(http.MethodPost, server.URL+"/api", nil)
			if err == nil {
				err = client.Send(req, nil)
			}
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("token calls = %d, want 1", got)
	}
	if got := apiCalls.Load(); got != goroutines {
		t.Fatalf("API calls = %d, want %d", got, goroutines)
	}
}

func TestPayloadLoggingIsOptInAndRedactsSensitiveFields(t *testing.T) {
	const cardNumber = "4111111111111111"
	payload := []byte(`{"CreditCard":{"TrackTwoData":"` + cardNumber + `"},"Result":"approved"}`)

	disabled := fmt.Sprint(bodyLogValue(payload, false))
	if strings.Contains(disabled, cardNumber) || !strings.Contains(disabled, "payload logging is disabled") {
		t.Fatalf("disabled payload log = %q", disabled)
	}

	enabled := fmt.Sprint(bodyLogValue(payload, true))
	if strings.Contains(enabled, cardNumber) {
		t.Fatalf("enabled payload log leaked card number: %q", enabled)
	}
	if !strings.Contains(enabled, "<redacted>") || !strings.Contains(enabled, "approved") {
		t.Fatalf("enabled payload log = %q, want redaction and non-sensitive data", enabled)
	}
}

func TestRequestLoggingDoesNotConsumeNonReplayableBody(t *testing.T) {
	body := io.NopCloser(strings.NewReader(`{"value":"kept"}`))
	req, err := http.NewRequest(http.MethodPost, "https://example.com", body)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	client.SetPayloadLogging(true)

	client.logHTTPRequest(req)
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"value":"kept"}`; got != want {
		t.Fatalf("body after logging = %q, want %q", got, want)
	}
}

func TestReadResponseBodyPropagatesReadError(t *testing.T) {
	want := errors.New("read failed")
	_, err := readResponseBody(errorReader{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapped %v", err, want)
	}
}

func clientWithToken(httpClient *http.Client) *Client {
	return &Client{
		Client: httpClient,
		token: &tokenResponse{
			AccessToken: "valid",
			expireTime:  time.Now().Add(time.Hour),
		},
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
