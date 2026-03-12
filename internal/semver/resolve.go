package semver

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

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
		v, err := semver.NewVersion(s)
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
	if qv, err := semver.NewVersion(query); err == nil {
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
