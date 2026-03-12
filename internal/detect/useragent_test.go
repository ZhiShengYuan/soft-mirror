package detect_test

import (
	"testing"

	"file-host/internal/detect"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		wantOS    string
		wantArch  string
	}{
		// OS detection
		{
			name:      "darwin via macintosh",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			wantOS:    "darwin",
			wantArch:  "amd64",
		},
		{
			name:      "darwin via darwin keyword",
			userAgent: "Go-http-client/1.1 (darwin; arm64)",
			wantOS:    "darwin",
			wantArch:  "arm64",
		},
		{
			name:      "darwin via macos keyword",
			userAgent: "curl/7.88.1 macos",
			wantOS:    "darwin",
			wantArch:  "amd64",
		},
		{
			name:      "windows",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			wantOS:    "windows",
			wantArch:  "amd64",
		},
		{
			name:      "windows via win keyword",
			userAgent: "some-client/1.0 win32",
			wantOS:    "windows",
			wantArch:  "amd64",
		},
		{
			name:      "linux explicit",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			wantOS:    "linux",
			wantArch:  "amd64",
		},
		// Arch detection
		{
			name:      "arm64 via aarch64",
			userAgent: "curl/7.88.1 aarch64-linux-gnu",
			wantOS:    "linux",
			wantArch:  "arm64",
		},
		{
			name:      "arm64 via arm64 keyword",
			userAgent: "Go-http-client/1.1 linux arm64",
			wantOS:    "linux",
			wantArch:  "arm64",
		},
		{
			name:      "amd64 via x86_64",
			userAgent: "wget/1.21 linux x86_64",
			wantOS:    "linux",
			wantArch:  "amd64",
		},
		{
			name:      "amd64 via amd64 keyword",
			userAgent: "custom-client amd64 linux",
			wantOS:    "linux",
			wantArch:  "amd64",
		},
		{
			name:      "amd64 via x64",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			wantOS:    "windows",
			wantArch:  "amd64",
		},
		// Defaults
		{
			name:      "empty user agent defaults to linux/amd64",
			userAgent: "",
			wantOS:    "linux",
			wantArch:  "amd64",
		},
		{
			name:      "unknown user agent defaults to linux/amd64",
			userAgent: "unknown-bot/1.0",
			wantOS:    "linux",
			wantArch:  "amd64",
		},
		// Case insensitivity
		{
			name:      "uppercase LINUX",
			userAgent: "LINUX X86_64",
			wantOS:    "linux",
			wantArch:  "amd64",
		},
		{
			name:      "mixed case Darwin",
			userAgent: "Darwin/21.0",
			wantOS:    "darwin",
			wantArch:  "amd64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOS, gotArch := detect.Detect(tt.userAgent)
			if gotOS != tt.wantOS {
				t.Errorf("Detect(%q) OS = %q, want %q", tt.userAgent, gotOS, tt.wantOS)
			}
			if gotArch != tt.wantArch {
				t.Errorf("Detect(%q) Arch = %q, want %q", tt.userAgent, gotArch, tt.wantArch)
			}
		})
	}
}
