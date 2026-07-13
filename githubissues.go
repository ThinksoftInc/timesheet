package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ghIssue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	PullRequest *struct{} `json:"pull_request"` // non-nil when the item is a PR
}

// githubToken reads ~/.gitcreds and returns the password/token for github.com,
// or an empty string if none is found or the file cannot be read.
// The file format is one credential URL per line:
//
//	https://user:token@github.com
func githubToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(home, ".gitcreds"))
	if err != nil {
		return ""
	}
	// explicitly ignore Close error to satisfy errcheck
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			continue
		}
		if strings.EqualFold(u.Hostname(), "github.com") {
			if pw, ok := u.User.Password(); ok && pw != "" {
				return pw
			}
		}
	}
	return ""
}

// githubIssues fetches open issues (excluding pull requests) for the given
// GitHub repo URL, e.g. "https://github.com/owner/repo".
// It authenticates using the token from ~/.gitcreds when available.
// Returns a non-nil error if the request fails or the API returns a non-200 status.
// githubIssues fetches open issues (excluding pull requests) for the given
// GitHub repo URL, e.g. "https://github.com/owner/repo". It authenticates
// using the token from ~/.gitcreds when available. The request uses the
// provided context for cancellation and timeouts; if ctx is nil,
// context.Background() is used.
func githubIssues(ctx context.Context, repoURL string, state string) ([]string, error) {
	if repoURL == "" {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	path := strings.TrimPrefix(repoURL, "https://github.com/")
	path = strings.TrimPrefix(path, "http://github.com/")
	apiURL := "https://api.github.com/repos/" + path + "/issues?state=" + state + "&per_page=100"

	if dbg != nil {
		dbg.Printf("githubIssues: fetching %s (token=%s)", apiURL, boolStr(githubToken() != ""))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		if dbg != nil {
			dbg.Printf("githubIssues: NewRequest failed: %v", err)
		}
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "timesheet-app")
	if token := githubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if dbg != nil {
			dbg.Printf("githubIssues: HTTP request failed: %v", err)
		}
		return nil, err
	}
	// explicitly ignore Close error to satisfy errcheck
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GitHub API: %s", resp.Status)
		if dbg != nil {
			dbg.Printf("githubIssues: %v", err)
		}
		return nil, err
	}

	var issues []ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		if dbg != nil {
			dbg.Printf("githubIssues: decode failed: %v", err)
		}
		return nil, err
	}

	var titles []string
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue // skip pull requests
		}
		titles = append(titles, fmt.Sprintf("#%d %s", issue.Number, issue.Title))
	}
	if dbg != nil {
		dbg.Printf("githubIssues: got %d issues (%d total items)", len(titles), len(issues))
	}
	return titles, nil
}
