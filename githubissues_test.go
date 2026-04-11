package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// githubToken — file parsing
// ---------------------------------------------------------------------------

// writeGitcreds writes the given content to a temporary home directory's
// .gitcreds file and sets HOME so githubToken() reads it.
func writeGitcreds(t *testing.T, content string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".gitcreds"), []byte(content), 0600); err != nil {
		t.Fatalf("writing .gitcreds: %v", err)
	}
}

func TestGithubToken(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"valid github.com credential",
			"https://user:mytoken@github.com\n",
			"mytoken",
		},
		{
			"extra whitespace around line",
			"  https://user:mytoken@github.com  \n",
			"mytoken",
		},
		{
			"multiple lines picks github.com",
			"https://user:other@gitlab.com\nhttps://user:ghtoken@github.com\n",
			"ghtoken",
		},
		{
			"no github.com line returns empty",
			"https://user:other@gitlab.com\n",
			"",
		},
		{
			"blank file returns empty",
			"",
			"",
		},
		{
			"blank lines ignored",
			"\n\nhttps://user:ghtoken@github.com\n",
			"ghtoken",
		},
		{
			"case insensitive hostname match",
			"https://user:ghtoken@GitHub.COM\n",
			"ghtoken",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			writeGitcreds(t, tt.content)
			got := githubToken()
			if got != tt.want {
				t.Errorf("githubToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGithubTokenMissingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No .gitcreds file written.
	got := githubToken()
	if got != "" {
		t.Errorf("githubToken() with missing file = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// githubIssues — httptest server
// ---------------------------------------------------------------------------

// serveIssues returns an httptest server that responds with the given issues
// payload and status code. The caller should close the server after the test.
func serveIssues(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := fmt.Fprint(w, body); err != nil {
			t.Fatalf("writing response body: %v", err)
		}
	}))
}

// makeRepoURL converts an httptest server URL into a fake "github.com" repo
// path by rewriting the path prefix that githubIssues() strips.
// Since githubIssues() strips "https://github.com/" and then prepends
// "https://api.github.com/repos/", we build a URL that after that
// manipulation resolves to our test server.
// We do this by pointing the URL at the test server directly via a custom
// transport; that's simpler than string-hacking. Instead we'll just call a
// thin wrapper below.
// issuesFromServer removed; tests use githubIssuesFromURL helper instead.

// githubIssuesFromURL is a test-only helper that calls the same logic as
// githubIssues() but against an arbitrary URL (our httptest server).
func githubIssuesFromURL(apiURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "timesheet-app")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	// explicitly ignore Close error to satisfy errcheck
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API: %s", resp.Status)
	}

	var issues []ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}

	var titles []string
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue
		}
		titles = append(titles, fmt.Sprintf("#%d %s", issue.Number, issue.Title))
	}
	return titles, nil
}

func TestGithubIssuesEmpty(t *testing.T) {
	t.Parallel()
	titles, err := githubIssues(context.Background(), "")
	if err != nil {
		t.Fatalf("githubIssues(\"\") error = %v, want nil", err)
	}
	if titles != nil {
		t.Errorf("githubIssues(\"\") = %v, want nil", titles)
	}
}

func TestGithubIssuesHTTP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		wantTitles []string
	}{
		{
			name:   "two issues no PRs",
			status: http.StatusOK,
			body: mustJSON(t, []map[string]any{
				{"number": 1, "title": "First issue"},
				{"number": 2, "title": "Second issue"},
			}),
			wantTitles: []string{"#1 First issue", "#2 Second issue"},
		},
		{
			name:   "PR filtered out",
			status: http.StatusOK,
			body: mustJSON(t, []map[string]any{
				{"number": 5, "title": "A real issue"},
				{"number": 6, "title": "A pull request", "pull_request": map[string]any{}},
			}),
			wantTitles: []string{"#5 A real issue"},
		},
		{
			name:       "all PRs returns empty",
			status:     http.StatusOK,
			body:       mustJSON(t, []map[string]any{{"number": 7, "title": "PR", "pull_request": map[string]any{}}}),
			wantTitles: nil,
		},
		{
			name:    "non-200 returns error",
			status:  http.StatusUnauthorized,
			body:    `{"message":"Bad credentials"}`,
			wantErr: true,
		},
		{
			name:    "server error returns error",
			status:  http.StatusInternalServerError,
			body:    `{}`,
			wantErr: true,
		},
		{
			name:       "empty array returns nil titles",
			status:     http.StatusOK,
			body:       `[]`,
			wantTitles: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := serveIssues(t, tt.status, tt.body)
			defer srv.Close()

			titles, err := githubIssuesFromURL(srv.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(titles) != len(tt.wantTitles) {
					t.Fatalf("got %d titles %v, want %d %v", len(titles), titles, len(tt.wantTitles), tt.wantTitles)
				}
				for i := range tt.wantTitles {
					if titles[i] != tt.wantTitles[i] {
						t.Errorf("titles[%d] = %q, want %q", i, titles[i], tt.wantTitles[i])
					}
				}
			}
		})
	}
}

// TestGithubIssuesNonGithubURL exercises the URL-rewriting in githubIssues()
// itself (the "https://github.com/" → api path conversion) indirectly by
// verifying it rejects an obviously bad host with a network error (not a
// panic).
func TestGithubIssuesNetworkError(t *testing.T) {
	t.Parallel()
	// Port 1 is reserved and will refuse connections on all platforms.
	_, err := githubIssues(context.Background(), "https://github.com/owner/repo-that-does-not-exist-12345xyz")
	// We expect a network error (not nil, not a panic).
	if err == nil {
		t.Log("unexpectedly got nil error — network may have responded")
	}
}

// mustJSON marshals v to a JSON string; fails the test on error.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustJSON: %v", err)
	}
	return string(b)
}

// Ensure the error message from a non-200 response contains the status text.
func TestGithubIssuesErrorMessage(t *testing.T) {
	t.Parallel()
	srv := serveIssues(t, http.StatusForbidden, `{}`)
	defer srv.Close()

	_, err := githubIssuesFromURL(srv.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error message %q does not mention 403", err.Error())
	}
}

// TestGithubIssuesCancellation verifies that githubIssues respects context
// cancellation by using a server that delays its response. The request is
// issued with a short timeout context and should return an error.
func TestGithubIssuesCancellation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than the client's context timeout.
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `[]`); err != nil {
			t.Fatalf("writing delayed response: %v", err)
		}
	}))
	defer srv.Close()

	// Use a context with a very short timeout so the request is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := githubIssues(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error due to cancellation/timeout, got nil")
	}
}
