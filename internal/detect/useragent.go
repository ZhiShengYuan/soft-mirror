package detect

import "strings"

// Detect returns os and arch from a User-Agent string.
func Detect(userAgent string) (os, arch string) {
	ua := strings.ToLower(userAgent)

	switch {
	case strings.Contains(ua, "darwin"), strings.Contains(ua, "macos"), strings.Contains(ua, "macintosh"):
		os = "darwin"
	case strings.Contains(ua, "windows"), strings.Contains(ua, "win"):
		os = "windows"
	case strings.Contains(ua, "linux"):
		os = "linux"
	default:
		os = "linux"
	}

	switch {
	case strings.Contains(ua, "aarch64"), strings.Contains(ua, "arm64"):
		arch = "arm64"
	case strings.Contains(ua, "x86_64"), strings.Contains(ua, "amd64"), strings.Contains(ua, "x64"):
		arch = "amd64"
	default:
		arch = "amd64"
	}

	return os, arch
}
