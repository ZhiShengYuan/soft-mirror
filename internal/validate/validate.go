package validate

import (
	"fmt"
	"regexp"
	"strings"

	"file-host/internal/model"

	"github.com/Masterminds/semver/v3"
)

var programNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// fourComponentVersionRegex matches versions like "2.1.39.0" (UWP-style).
var fourComponentVersionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

// ValidateProgramName checks that name matches ^[a-zA-Z0-9_-]+$ and is at most 128 characters.
func ValidateProgramName(name string) error {
	if name == "" {
		return fmt.Errorf("program name must not be empty")
	}
	if len(name) > 128 {
		return fmt.Errorf("program name must be at most 128 characters, got %d", len(name))
	}
	if !programNameRegex.MatchString(name) {
		return fmt.Errorf("program name %q contains invalid characters; only alphanumeric, hyphens, and underscores are allowed", name)
	}
	return nil
}

// ValidateVersion checks that version is valid semver (with or without "v" prefix)
// or a 4-component version as used by UWP packages (e.g. "2.1.39.0").
func ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version must not be empty")
	}
	v := strings.TrimPrefix(version, "v")
	if _, err := semver.NewVersion(v); err == nil {
		return nil
	}
	if fourComponentVersionRegex.MatchString(v) {
		return nil
	}
	return fmt.Errorf("invalid version %q: must be semver (e.g. 1.2.3) or 4-component (e.g. 1.2.3.0)", version)
}

// ValidateOS checks that os is in the allowed OS list.
func ValidateOS(os string) error {
	if !model.AllowedOS[os] {
		return fmt.Errorf("unsupported OS %q; allowed values: linux, windows, darwin", os)
	}
	return nil
}

// ValidateArch checks that arch is in the allowed architecture list.
func ValidateArch(arch string) error {
	if !model.AllowedArch[arch] {
		return fmt.Errorf("unsupported architecture %q; allowed values: amd64, arm64", arch)
	}
	return nil
}

// ValidatePlatform validates both the OS and architecture.
func ValidatePlatform(os, arch string) error {
	if err := ValidateOS(os); err != nil {
		return err
	}
	if err := ValidateArch(arch); err != nil {
		return err
	}
	return nil
}
