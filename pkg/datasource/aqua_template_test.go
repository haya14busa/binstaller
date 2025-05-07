package datasource

import "testing"

func TestConvertAquaTemplateToInstallSpec(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:     "simple variables",
			input:    "{{.Version}}_{{.OS}}_{{.Arch}}.{{.Format}}",
			expected: "${TAG}_${OS}_${ARCH}${EXT}",
		},
		{
			name:     "function call trimV",
			input:    "reviewdog_{{trimV .Version}}_{{.OS}}_{{.Arch}}.{{.Format}}",
			expected: "reviewdog_${VERSION}_${OS}_${ARCH}${EXT}",
		},
		{
			name:      "function call tolower",
			input:     "{{tolower .OS}}_{{.Version}}_{{.Arch}}.{{.Format}}",
			expectErr: true,
		},
		{
			name:     "mixed whitespace",
			input:    "  {{ .Version }}_{{ .OS }}_{{ .Arch }}.{{ .Format }}  ",
			expected: "${TAG}_${OS}_${ARCH}${EXT}",
		},
		{
			name:     "semver variable",
			input:    "{{.SemVer}}-{{.OS}}-{{.Arch}}.{{.Format}}",
			expected: "${VERSION}-${OS}-${ARCH}${EXT}",
		},
		{
			name:      "unknown function",
			input:     "{{unknown .Version}}_{{.OS}}_{{.Arch}}.{{.Format}}",
			expectErr: true,
		},
		{
			name:     "no extension in template",
			input:    "{{.Version}}_{{.OS}}_{{.Arch}}",
			expected: "${TAG}_${OS}_${ARCH}",
		},
		{
			name:     "already ends with .Format",
			input:    "{{.Version}}_{{.OS}}_{{.Arch}}.{{.Format}}",
			expected: "${TAG}_${OS}_${ARCH}${EXT}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertAquaTemplateToInstallSpec(tt.input, nil)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("ConvertAquaTemplateToInstallSpec(%q) = %q, want %q", tt.input, got, tt.expected)
				}
			}
		})
	}
}
