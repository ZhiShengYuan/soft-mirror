package semver

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// coerceVersion parses s as semver. If s has 4 dot-separated components
// (e.g. UWP "2.1.39.0"), it falls back to parsing the first three components.
// The caller always uses the original string as the on-disk name.
func coerceVersion(s string) (*semver.Version, error) {
	v := strings.TrimPrefix(s, "v")
	if parsed, err := semver.NewVersion(v); err == nil {
		return parsed, nil
	}
	if parts := strings.Split(v, "."); len(parts) == 4 {
		return semver.NewVersion(strings.Join(parts[:3], "."))
	}
	return nil, fmt.Errorf("cannot parse version %q", s)
}

// Resolve returns the best version string from available versions given a query.
// Query can be:
//
//	"latest" -> highest non-prerelease version; if none exist, highest overall
//	"v1.2.3" or "1.2.3" exact -> return it if present in available list (compare normalized)
//	semver constraint ("^1.0", "~1.2", ">=1.0 <2.0") -> highest matching version
//
// Returns error if no match found or query is invalid.
type parsed struct {
	original string
	version  *semver.Version
}

func Resolve(query string, available []string) (string, error) {
	var versions []parsed
	for _, s := range available {
		v, err := coerceVersion(s)
		if err != nil {
			continue
		}
		versions = append(versions, parsed{original: s, version: v})
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no valid versions in available list")
	}

	if query == "latest" {
		return resolveLatest(versions)
	}

	// Try exact match first.
	if qv, err := coerceVersion(query); err == nil {
		for _, p := range versions {
			if p.version.Equal(qv) {
				return p.original, nil
			}
		}
		return "", fmt.Errorf("version %s not found in available list", query)
	}

	// Try constraint match.
	c, err := semver.NewConstraint(query)
	if err != nil {
		return "", fmt.Errorf("invalid version query %q: %w", query, err)
	}

	var best *parsed
	for i := range versions {
		if c.Check(versions[i].version) {
			if best == nil || versions[i].version.GreaterThan(best.version) {
				best = &versions[i]
			}
		}
	}

	if best == nil {
		return "", fmt.Errorf("no version matching constraint %q", query)
	}
	return best.original, nil
}

func resolveLatest(versions []parsed) (string, error) {
	var best *parsed
	// Prefer non-prerelease versions.
	for i := range versions {
		if versions[i].version.Prerelease() == "" {
			if best == nil || versions[i].version.GreaterThan(best.version) {
				best = &versions[i]
			}
		}
	}
	if best != nil {
		return best.original, nil
	}
	// Fall back to highest overall.
	for i := range versions {
		if best == nil || versions[i].version.GreaterThan(best.version) {
			best = &versions[i]
		}
	}
	return best.original, nil
}
