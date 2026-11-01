package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	"jakapil-cli/internal/authentication"
)

func TestAPStatus(t *testing.T) {
	if status, ok := apiStatus(nil); ok || status != 0 {
		t.Fatalf("nil error must not be an API error, got status %d ok %v", status, ok)
	}

	// A gophercloud-style error reports its HTTP status.
	apiErr := gophercloud.ErrUnexpectedResponseCode{Actual: 404}
	if status, ok := apiStatus(apiErr); !ok || status != 404 {
		t.Fatalf("expected 404 API status, got status %d ok %v", status, ok)
	}

	// Wrapping with %w keeps the failure detectable (mks-go/craas-go do this).
	if status, ok := apiStatus(fmt.Errorf("list: %w", apiErr)); !ok || status != 404 {
		t.Fatalf("wrapped API error not detected, got status %d ok %v", status, ok)
	}

	// Statuses below 400 (e.g. 3xx) are still returned as-is, so callers can
	// decide which range to treat as a hard API failure.
	if status, ok := apiStatus(gophercloud.ErrUnexpectedResponseCode{Actual: 302}); !ok || status != 302 {
		t.Fatalf("expected status 302, got status %d ok %v", status, ok)
	}

	// A non-API error is not an API error.
	if _, ok := apiStatus(errors.New("boom")); ok {
		t.Fatal("plain error must not be reported as an API error")
	}
}

func TestHandleError(t *testing.T) {
	// API 4xx/5xx: only the message, no usage dump.
	for _, status := range []int{400, 401, 404, 429, 500, 503} {
		msg, showUsage := handleError(gophercloud.ErrUnexpectedResponseCode{Actual: status})
		if showUsage {
			t.Fatalf("status %d must not carry usage, got %q", status, msg)
		}
		if msg == "" {
			t.Fatalf("status %d must still render a message", status)
		}
	}

	// Non-API errors keep the usage block for the user to fix the invocation.
	msg, showUsage := handleError(errors.New("accepts 1 arg(s), received 0"))
	if !showUsage || msg == "" {
		t.Fatalf("plain error must keep usage, got %q showUsage %v", msg, showUsage)
	}

	// A locally detected expired stored token is reported without usage: the
	// message already tells the user to renew it, so no invocation help.
	expired := fmt.Errorf("stored token has expired at 2026-01-01T00:00:00Z: run \"jakapil-cli authenticate token issue\" to renew it: %w", authentication.ErrTokenExpired)
	msg, showUsage = handleError(expired)
	if showUsage || msg == "" {
		t.Fatalf("expired token must not carry usage, got %q showUsage %v", msg, showUsage)
	}
}
