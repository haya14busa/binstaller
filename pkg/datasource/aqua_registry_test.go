package datasource

import (
	"testing"
)

func TestIsVersionConstraintSatisfiedForLatest(t *testing.T) {
	tests := []struct {
		constraint string
		want       bool
	}{
		{"", false},
		{"true", true},
		{"false", false},
		{`semver(">= 0.4.0")`, true},
		{`semver("< 0.4.0")`, false},
		{`semverWithVersion(">= 4.2.0", trimPrefix(Version, "kustomize/"))`, true},
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			got := isVersionConstraintSatisfiedForLatest(tt.constraint)
			if got != tt.want {
				t.Errorf("isVersionConstraintSatisfiedForLatest(%q) = %v, want %v", tt.constraint, got, tt.want)
			}
		})
	}
}
