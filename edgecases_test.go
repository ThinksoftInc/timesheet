package main

import "testing"

func TestParseDurationEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want float64
	}{
		{"unicode digits", "٠1:30", 0},
		{"very large hours", "1000000:00", 1000000.0},
		{"leading plus", "+01:30", 1.5},
		{"leading zeros", "001:05", 1 + 5.0/60.0},
		{"non-digit hours", "1a:05", 0},
		{"unicode colon", "01：30", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseDuration(tt.in)
			if got != tt.want {
				t.Fatalf("parseDuration(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseGitURLEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"multiple entries", "giturl: https://first.com/repo giturl: https://second.com/repo", "https://first.com/repo"},
		{"percent encoded", "giturl: https://example.com/a%20b", "https://example.com/a%20b"},
		{"quoted url returns empty", "giturl: \"https://example.com/repo\"", ""},
		{"url with html before", "<a href=\"x\">link</a> giturl: https://example.com/x/", "https://example.com/x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseGitURL(tt.in)
			if got != tt.want {
				t.Fatalf("parseGitURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
