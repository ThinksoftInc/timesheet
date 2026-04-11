package main

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// parseDuration
// ---------------------------------------------------------------------------

func TestParseDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"half hour", "00:30", 0.5},
		{"one hour", "01:00", 1.0},
		{"one and a half hours", "01:30", 1.5},
		{"two hours", "02:00", 2.0},
		{"eight hours", "08:00", 8.0},
		{"max 24 hours", "24:00", 24.0},
		{"no colon", "0130", 0},
		{"empty string", "", 0},
		{"only colon", ":", 0},
		{"spaces around", " 01:30 ", 1.5},
		{"single-digit minute", "1:05", 1 + 5.0/60.0},
		{"invalid minutes", "01:60", 0},
		{"negative numbers", "-1:30", 0},
		{"minutes negative", "01:-5", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseDuration(tt.input)
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// durationOptions
// ---------------------------------------------------------------------------

func TestDurationOptions(t *testing.T) {
	t.Parallel()
	opts := durationOptions()

	if len(opts) != 48 {
		t.Errorf("len(durationOptions()) = %d, want 48", len(opts))
	}
	if opts[0] != "00:30" {
		t.Errorf("first option = %q, want %q", opts[0], "00:30")
	}
	if opts[len(opts)-1] != "24:00" {
		t.Errorf("last option = %q, want %q", opts[len(opts)-1], "24:00")
	}
	// Every entry should be parseable to a positive value.
	for _, o := range opts {
		if v := parseDuration(o); v <= 0 {
			t.Errorf("parseDuration(%q) = %v, want > 0", o, v)
		}
	}
}

// ---------------------------------------------------------------------------
// dateOptions
// ---------------------------------------------------------------------------

func TestDateOptions(t *testing.T) {
	t.Parallel()
	dates, idx := dateOptions()

	if idx != 0 {
		t.Errorf("dateOptions() default index = %d, want 0", idx)
	}
	if len(dates) == 0 {
		t.Fatal("dateOptions() returned empty slice")
	}
	// Dates must be in descending order (newest first).
	for i := 1; i < len(dates); i++ {
		if dates[i] >= dates[i-1] {
			t.Errorf("dates not descending at index %d: %q >= %q", i, dates[i], dates[i-1])
		}
	}
	// All entries should look like YYYY-MM-DD.
	for _, d := range dates {
		if len(d) != 10 || d[4] != '-' || d[7] != '-' {
			t.Errorf("date %q does not match YYYY-MM-DD format", d)
		}
	}
}

// ---------------------------------------------------------------------------
// formatDescription
// ---------------------------------------------------------------------------

func TestFormatDescription(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		issueText string
		areaText  string
		want      string
	}{
		// No issue, non-empty text.
		{"plain text", "", "fixed login bug", "fixed login bug"},
		// No issue, whitespace-only text → "/".
		{"empty text becomes slash", "", "   ", "/"},
		// No issue, empty text → "/".
		{"empty text", "", "", "/"},
		// Issue with text.
		{"issue with text", "#42 Some bug title", "did the work", "#42 - did the work"},
		// Issue with empty text → "/" as description.
		{"issue empty text", "#7 Another issue", "  ", "#7 - /"},
		// Issue number only (no space after number) — treated as whole number.
		{"issue no space", "#99", "work", "#99 - work"},
		// Non-issue issue text (no leading #) → ignored, plain text returned.
		{"no hash prefix", "42 Some title", "work", "work"},
		// Leading "#" with no number → ignored.
		{"hash no number", "#abc title", "work", "work"},
		// Issue text with leading/trailing spaces in area text.
		{"trims area text", "#10 Title", "  trimmed  ", "#10 - trimmed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDescription(tt.issueText, tt.areaText)
			if got != tt.want {
				t.Errorf("formatDescription(%q, %q) = %q, want %q",
					tt.issueText, tt.areaText, got, tt.want)
			}
		})
	}
}

// Ensure formatDescription never returns an empty string.
func TestFormatDescriptionNeverEmpty(t *testing.T) {
	t.Parallel()
	inputs := []struct{ issue, area string }{
		{"", ""},
		{"", "   "},
		{"#1 x", ""},
		{"#1 x", "   "},
	}
	for _, tt := range inputs {
		got := formatDescription(tt.issue, tt.area)
		if strings.TrimSpace(got) == "" {
			t.Errorf("formatDescription(%q, %q) returned empty/blank string %q", tt.issue, tt.area, got)
		}
	}
}
