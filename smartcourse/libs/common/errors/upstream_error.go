package errors

import "fmt"

// UpstreamError is a failure talking to another service (network error or
// unexpected response), as opposed to a business-rule rejection.
type UpstreamError struct {
	Service string
	Err     error
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream %s error: %v", e.Service, e.Err)
}

func (e *UpstreamError) Unwrap() error {
	return e.Err
}
