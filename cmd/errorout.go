package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"jakapil-cli/internal/authentication"
)

// ANSI colour sequences used to highlight command errors on a terminal. They
// are only emitted when stderr is a TTY; redirected/piped output stays plain so
// it can be parsed and logged without escape codes.
const (
	ansiRed   = "\x1b[1;31m" // bold red
	ansiReset = "\x1b[0m"
)

// apiStatus returns the HTTP status a command error reported from the APIs the
// CLI talks to (Keystone, VPC/gophercloud, MKS and CRaaS SDKs all surface
// non-2xx responses through an error exposing GetStatusCode()), or 0/false when
// the error is not an API error at all. Walking the wrap chain keeps errors that
// are wrapped with %w detectable.
func apiStatus(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var sc interface{ GetStatusCode() int }
	if errors.As(err, &sc) {
		return sc.GetStatusCode(), true
	}
	return 0, false
}

// handleError decides how a command error is reported to the user.
//
// Two kinds of failures are reported with just the message — the full usage
// block is dropped because there is nothing to fix in the invocation itself:
//
//   - API errors (an HTTP status of 400 or higher): the endpoint refused the
//     request;
//   - a locally detected expired stored token (ErrTokenExpired): the user is
//     already told to renew it, so dumping the command usage is noise.
//
// For any other error (bad CLI usage, missing arguments, file or network
// problems) cobra's default behaviour is preserved and the usage block
// accompanies the message so the user sees how to fix the invocation.
//
// It returns the message to print and whether the usage text should follow.
func handleError(c *cobra.Command, err error) (msg string, showUsage bool) {
	msg = "Error: " + err.Error()
	if status, ok := apiStatus(err); ok && status >= 400 {
		return msg, false
	}
	if errors.Is(err, authentication.ErrTokenExpired) {
		return msg, false
	}
	return msg, true
}

// colorize wraps s in ANSI red when stderr is attached to a terminal and returns
// s unchanged otherwise.
func colorize(s string) string {
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return ansiRed + s + ansiReset
	}
	return s
}
