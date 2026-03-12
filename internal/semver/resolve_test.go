package semver

import (
	"testing"
)

func TestResolveLatest(t *testing.T) {
	versions := []string{"v1.0.0", "v1.1.0", "v2.0.0-beta.1"}
	got, err := Resolve("latest", versions)
	if err != nil {
		t.Fatalf("Resolve latest: %v", err)
	}
	if got != "v1.1.0" {
		t.Errorf("expected v1.1.0, got %s", got)
	}
}

func TestResolveLatestFallbackToPrerelease(t *testing.T) {
	versions := []string{"v2.0.0-beta.1", "v2.0.0-alpha.1"}
	got, err := Resolve("latest", versions)
	if err != nil {
		t.Fatalf("Resolve latest prerelease fallback: %v", err)
	}
	if got != "v2.0.0-beta.1" {
		t.Errorf("expected v2.0.0-beta.1, got %s", got)
	}
}

func TestResolveExact(t *testing.T) {
	versions := []string{"v1.0.0", "v1.1.0", "v2.0.0-beta.1"}
	got, err := Resolve("v2.0.0-beta.1", versions)
	if err != nil {
		t.Fatalf("Resolve exact: %v", err)
	}
	if got != "v2.0.0-beta.1" {
		t.Errorf("expected v2.0.0-beta.1, got %s", got)
	}
}

func TestResolveExactNotFound(t *testing.T) {
	versions := []string{"v1.0.0", "v1.1.0"}
	_, err := Resolve("v9.9.9", versions)
	if err == nil {
		t.Error("expected error for missing exact version, got nil")
	}
}

func TestResolveConstraint(t *testing.T) {
	versions := []string{"v1.0.0", "v1.1.0", "v2.0.0"}
	got, err := Resolve("^1.0", versions)
	if err != nil {
		t.Fatalf("Resolve constraint: %v", err)
	}
	if got != "v1.1.0" {
		t.Errorf("expected v1.1.0 for ^1.0, got %s", got)
	}
}

func TestResolveConstraintRange(t *testing.T) {
	versions := []string{"v1.0.0", "v1.1.0", "v2.0.0"}
	got, err := Resolve(">=1.0 <2.0", versions)
	if err != nil {
		t.Fatalf("Resolve range: %v", err)
	}
	if got != "v1.1.0" {
		t.Errorf("expected v1.1.0, got %s", got)
	}
}

func TestResolveNoVersions(t *testing.T) {
	_, err := Resolve("latest", []string{})
	if err == nil {
		t.Error("expected error for empty list, got nil")
	}
}
