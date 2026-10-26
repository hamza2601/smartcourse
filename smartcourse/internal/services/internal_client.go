package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	apperrors "smartcourse/internal/errors"
)

const (
	internalCallTimeout  = 5 * time.Second
	maxInternalBodyBytes = 1 << 20
	userServiceName      = "user-service"
	courseServiceName    = "course-service"
)

func newInternalHTTPClient() *http.Client {
	return &http.Client{Timeout: internalCallTimeout}
}

// fetchInternal GETs url and decodes a 200 JSON body into out. A 404 is a
// business answer (found=false, no error); network failures, other statuses
// and unparseable bodies are returned as UpstreamError.
func fetchInternal(ctx context.Context, client *http.Client, service, url string, out any) (found bool, err error) {
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

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxInternalBodyBytes)).Decode(out); err != nil {
		return false, &apperrors.UpstreamError{Service: service, Err: fmt.Errorf("invalid response body: %w", err)}
	}
	return true, nil
}
