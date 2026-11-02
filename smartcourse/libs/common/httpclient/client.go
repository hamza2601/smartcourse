package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
)

const (
	callTimeout  = 5 * time.Second
	maxBodyBytes = 1 << 20
)

// NewClient returns a client for service-to-service calls; every request it
// sends carries the internal API key in X-Internal-Token.
func NewClient(internalAPIKey string) *http.Client {
	return &http.Client{
		Timeout:   callTimeout,
		Transport: &internalKeyTransport{apiKey: internalAPIKey, base: http.DefaultTransport},
	}
}

type internalKeyTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *internalKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// A RoundTripper must not modify the caller's request.
	req = req.Clone(req.Context())
	req.Header.Set(auth.InternalTokenHeader, t.apiKey)
	return t.base.RoundTrip(req)
}

// FetchInternal GETs url and decodes a 200 JSON body into out. A 404 is a
// business answer (found=false, no error); network failures, other statuses
// and unparseable bodies are returned as UpstreamError.
func FetchInternal(ctx context.Context, client *http.Client, service, url string, out any) (found bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, &apperrors.UpstreamError{Service: service, Err: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, &apperrors.UpstreamError{Service: service, Err: err}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return false, nil
	case resp.StatusCode != http.StatusOK:
		return false, &apperrors.UpstreamError{Service: service, Err: fmt.Errorf("unexpected status %d", resp.StatusCode)}
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(out); err != nil {
		return false, &apperrors.UpstreamError{Service: service, Err: fmt.Errorf("invalid response body: %w", err)}
	}
	return true, nil
}
