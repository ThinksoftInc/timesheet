package main

import (
	"testing"
)

// ---------------------------------------------------------------------------
// stripHTML
// ---------------------------------------------------------------------------

func TestStripHTML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text unchanged", "hello world", "hello world"},
		{"single tag removed", "<p>hello</p>", "hello"},
		{"nested tags removed", "<div><b>bold</b></div>", "bold"},
		{"HTML entity decoded", "&amp;", "&"},
		{"nbsp replaced with space", "a\u00a0b", "a b"},
		{"nbsp entity replaced", "a&nbsp;b", "a b"},
		{"mixed html and entities", "<p>Tom &amp; Jerry</p>", "Tom & Jerry"},
		{"empty string", "", ""},
		{"only tags", "<br/><hr/>", ""},
		{"self-closing tag", "<img src=\"x\"/>text", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseGitURL
// ---------------------------------------------------------------------------

func TestParseGitURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			"plain giturl line",
			"giturl: https://github.com/owner/repo",
			"https://github.com/owner/repo",
		},
		{
			"trailing slash stripped",
			"giturl: https://github.com/owner/repo/",
			"https://github.com/owner/repo",
		},
		{
			"case insensitive label",
			"GitURL: https://github.com/owner/repo",
			"https://github.com/owner/repo",
		},
		{
			"url embedded in HTML",
			"<p>giturl: https://github.com/owner/repo</p>",
			"https://github.com/owner/repo",
		},
		{
			"url with path",
			"giturl: https://github.com/org/my-project",
			"https://github.com/org/my-project",
		},
		{
			"http scheme",
			"giturl: http://github.com/owner/repo",
			"http://github.com/owner/repo",
		},
		{
			"no giturl label returns empty",
			"just a plain description with no url",
			"",
		},
		{
			"empty description",
			"",
			"",
		},
		{
			"html encoded ampersand in surrounding text",
			"<p>Project info &amp; details</p><p>giturl: https://github.com/owner/repo</p>",
			"https://github.com/owner/repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseGitURL(tt.description)
			if got != tt.want {
				t.Errorf("parseGitURL(%q) = %q, want %q", tt.description, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// projectNames
// ---------------------------------------------------------------------------

func TestProjectNames(t *testing.T) {
	t.Parallel()
	infos := []projectInfo{
		{id: 1, name: "Alpha"},
		{id: 2, name: "Beta"},
		{id: 3, name: "Gamma"},
	}
	got := projectNames(infos)
	want := []string{"Alpha", "Beta", "Gamma"}
	if len(got) != len(want) {
		t.Fatalf("projectNames() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("projectNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestProjectNamesEmpty(t *testing.T) {
	t.Parallel()
	got := projectNames(nil)
	if len(got) != 0 {
		t.Errorf("projectNames(nil) = %v, want empty slice", got)
	}
}

// ---------------------------------------------------------------------------
// findProject
// ---------------------------------------------------------------------------

func TestFindProject(t *testing.T) {
	t.Parallel()
	infos := []projectInfo{
		{id: 1, name: "Alpha"},
		{id: 2, name: "Beta"},
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		p := findProject(infos, "Beta")
		if p == nil {
			t.Fatal("findProject returned nil, want non-nil")
		}
		if p.id != 2 {
			t.Errorf("findProject id = %d, want 2", p.id)
		}
	})

	t.Run("not found returns nil", func(t *testing.T) {
		t.Parallel()
		p := findProject(infos, "Gamma")
		if p != nil {
			t.Errorf("findProject returned %+v, want nil", p)
		}
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		t.Parallel()
		p := findProject(nil, "Alpha")
		if p != nil {
			t.Errorf("findProject on nil slice returned %+v, want nil", p)
		}
	})
}

// Ensure findProject returns a pointer into the original slice (not a copy).
func TestFindProjectReturnsMutablePointer(t *testing.T) {
	t.Parallel()
	infos := []projectInfo{{id: 1, name: "Alpha"}}
	p := findProject(infos, "Alpha")
	if p == nil {
		t.Fatal("findProject returned nil")
	}
	p.id = 99
	if infos[0].id != 99 {
		t.Errorf("modifying returned pointer did not affect original slice (got %d)", infos[0].id)
	}
}

// ---------------------------------------------------------------------------
// taskNames
// ---------------------------------------------------------------------------

func TestTaskNames(t *testing.T) {
	t.Parallel()
	tasks := []taskInfo{
		{id: 10, name: "Odoo Support"},
		{id: 20, name: "Development"},
	}
	got := taskNames(tasks)
	want := []string{"Odoo Support", "Development"}
	if len(got) != len(want) {
		t.Fatalf("taskNames() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("taskNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTaskNamesEmpty(t *testing.T) {
	t.Parallel()
	got := taskNames(nil)
	if len(got) != 0 {
		t.Errorf("taskNames(nil) = %v, want empty slice", got)
	}
}
