/*
Copyright © 2026 Kot1gruN
*/
package client

import "testing"

func TestV2Endpoint(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{"default v1", "https://cr.selcloud.ru/api/v1", "https://cr.selcloud.ru/api/v2"},
		{"already v2", "https://cr.selcloud.ru/api/v2", "https://cr.selcloud.ru/api/v2"},
		{"no version suffix", "https://cr.selcloud.ru/api", "https://cr.selcloud.ru/api/v2"},
		{"with trailing slash", "https://cr.selcloud.ru/api/v1/", "https://cr.selcloud.ru/api/v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v2Endpoint(tt.in); got != tt.expected {
				t.Fatalf("v2Endpoint(%q) = %q, want %q", tt.in, got, tt.expected)
			}
		})
	}
}
