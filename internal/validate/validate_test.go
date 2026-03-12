package validate

import (
	"strings"
	"testing"
)

func TestValidateProgramName(t *testing.T) {
	valid := []string{"myapp", "my-app", "my_app", "App123", "a", strings.Repeat("a", 128)}
	for _, name := range valid {
		if err := ValidateProgramName(name); err != nil {
			t.Errorf("expected valid name %q to pass, got: %v", name, err)
		}
	}

	invalid := []string{"", "my app", "my/app", "../evil", strings.Repeat("a", 129), "app!"}
	for _, name := range invalid {
		if err := ValidateProgramName(name); err == nil {
			t.Errorf("expected invalid name %q to fail, got nil", name)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	valid := []string{"v1.0.0", "1.0.0", "v1.2.3-beta.1", "v0.0.1"}
	for _, v := range valid {
		if err := ValidateVersion(v); err != nil {
			t.Errorf("expected valid version %q to pass, got: %v", v, err)
		}
	}

	invalid := []string{"", "latest", "not-a-version"}
	for _, v := range invalid {
		if err := ValidateVersion(v); err == nil {
			t.Errorf("expected invalid version %q to fail, got nil", v)
		}
	}
}

func TestValidateOS(t *testing.T) {
	for _, os := range []string{"linux", "windows", "darwin"} {
		if err := ValidateOS(os); err != nil {
			t.Errorf("expected valid OS %q to pass, got: %v", os, err)
		}
	}
	for _, os := range []string{"", "macos", "win", "freebsd"} {
		if err := ValidateOS(os); err == nil {
			t.Errorf("expected invalid OS %q to fail, got nil", os)
		}
	}
}

func TestValidateArch(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		if err := ValidateArch(arch); err != nil {
			t.Errorf("expected valid arch %q to pass, got: %v", arch, err)
		}
	}
	for _, arch := range []string{"", "x86", "386", "arm"} {
		if err := ValidateArch(arch); err == nil {
			t.Errorf("expected invalid arch %q to fail, got nil", arch)
		}
	}
}
