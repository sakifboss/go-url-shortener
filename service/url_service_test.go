package service

import (
	"net"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			input:   "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			input:   "http://example.com",
			wantErr: false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			input:   "example.com",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			input:   "ftp://example.com/file.txt",
			wantErr: true,
		},
		{
			name:    "missing host",
			input:   "https:///path",
			wantErr: true,
		},
		{
			name:    "URL with credentials",
			input:   "https://user:password@example.com",
			wantErr: true,
		},
		{
			name:    "loopback IPv4",
			input:   "http://127.0.0.1",
			wantErr: true,
		},
		{
			name:    "private IPv4",
			input:   "http://192.168.1.1",
			wantErr: true,
		},
		{
			name:    "private 10.x IPv4",
			input:   "http://10.0.0.1",
			wantErr: true,
		},
		{
			name:    "link-local IPv4",
			input:   "http://169.254.1.1",
			wantErr: true,
		},
		{
			name:    "loopback IPv6",
			input:   "http://[::1]",
			wantErr: true,
		},
		{
			name:    "public IPv4",
			input:   "http://8.8.8.8",
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateURL(test.input)

			if test.wantErr && err == nil {
				t.Fatalf(
					"expected error for %q, got nil",
					test.input,
				)
			}

			if !test.wantErr && err != nil {
				t.Fatalf(
					"expected no error for %q, got %v",
					test.input,
					err,
				)
			}
		})
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{
			name: "public IPv4",
			ip:   "8.8.8.8",
			want: true,
		},
		{
			name: "loopback IPv4",
			ip:   "127.0.0.1",
			want: false,
		},
		{
			name: "private IPv4",
			ip:   "192.168.1.1",
			want: false,
		},
		{
			name: "private 10.x IPv4",
			ip:   "10.0.0.1",
			want: false,
		},
		{
			name: "link-local IPv4",
			ip:   "169.254.1.1",
			want: false,
		},
		{
			name: "loopback IPv6",
			ip:   "::1",
			want: false,
		},
		{
			name: "unspecified IPv4",
			ip:   "0.0.0.0",
			want: false,
		},
		{
			name: "unspecified IPv6",
			ip:   "::",
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ip := net.ParseIP(test.ip)

			if ip == nil {
				t.Fatalf(
					"failed to parse test IP %q",
					test.ip,
				)
			}

			got := isPublicIP(ip)

			if got != test.want {
				t.Fatalf(
					"isPublicIP(%q) = %v, want %v",
					test.ip,
					got,
					test.want,
				)
			}
		})
	}
}
