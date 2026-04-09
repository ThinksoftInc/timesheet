package main

import (
	"html"
	"regexp"
	"slices"
	"strings"
)

type projectInfo struct {
	id          int
	name        string
	description string
	accountID   int
}

type taskInfo struct {
	id   int
	name string
}

var (
	gitURLRe  = regexp.MustCompile(`(?i)giturl:\s*(https?://[^\s<"]+)`)
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
)

// projects fetches all in-progress projects with their descriptions.
func projects() []projectInfo {
	conn, err := NewConn()
	if err != nil {
		return nil
	}
	records, err := conn.SearchRead("project.project", 0, 0,
		[]string{"id", "name", "description", "account_id"},
		[]any{[]any{"stage_id", "=", "In Progress"}})
	if err != nil {
		return nil
	}
	infos := make([]projectInfo, 0, len(records))
	for _, record := range records {
		id, _ := record["id"].(float64)
		name, _ := record["name"].(string)
		desc, _ := record["description"].(string)
		// account_id is a Many2one: Odoo returns [id, name] or false.
		var accountID int
		if v, ok := record["account_id"].([]any); ok && len(v) > 0 {
			if f, ok := v[0].(float64); ok {
				accountID = int(f)
			}
		}
		infos = append(infos, projectInfo{
			id:          int(id),
			name:        name,
			description: desc,
			accountID:   accountID,
		})
	}
	slices.SortFunc(infos, func(a, b projectInfo) int {
		return strings.Compare(a.name, b.name)
	})
	return infos
}

// stripHTML removes all HTML tags from s and decodes HTML entities,
// giving us plain text to parse.
func stripHTML(s string) string {
	clean := html.UnescapeString(htmlTagRe.ReplaceAllString(s, ""))
	return strings.ReplaceAll(clean, "\u00a0", " ")
}

// parseGitURL extracts the URL following a "giturl:" label from a project
// description. The description is typically an Odoo HTML field, so we strip
// tags first to get a clean text representation.
func parseGitURL(description string) string {
	plain := stripHTML(description)
	matches := gitURLRe.FindStringSubmatch(plain)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimRight(matches[1], "/")
}

func projectNames(infos []projectInfo) []string {
	names := make([]string, len(infos))
	for i, p := range infos {
		names[i] = p.name
	}
	return names
}

func findProject(infos []projectInfo, name string) *projectInfo {
	for i := range infos {
		if infos[i].name == name {
			return &infos[i]
		}
	}
	return nil
}

// projectTasks fetches tasks for the given project name, returning id and name
// for each so callers can pass IDs directly to Odoo without extra lookups.
func projectTasks(projectName string) []taskInfo {
	conn, err := NewConn()
	if err != nil {
		return nil
	}
	allowedTasks := []any{"Odoo Support", "Development", "Planning / Discovery"}
	nameFilter := []any{"name", "in", allowedTasks}
	var records []map[string]any
	if projectName != "" {
		records, err = conn.SearchRead("project.task", 0, 0, []string{"id", "name"},
			[]any{[]any{"project_id.name", "=", projectName}, nameFilter})
	} else {
		records, err = conn.SearchRead("project.task", 0, 0, []string{"id", "name"},
			[]any{nameFilter})
	}
	if err != nil {
		return nil
	}
	tasks := make([]taskInfo, 0, len(records))
	for _, record := range records {
		id, _ := record["id"].(float64)
		name, _ := record["name"].(string)
		tasks = append(tasks, taskInfo{id: int(id), name: name})
	}
	taskOrder := map[string]int{"Odoo Support": 0, "Planning / Discovery": 1, "Development": 2}
	slices.SortFunc(tasks, func(a, b taskInfo) int {
		return taskOrder[a.name] - taskOrder[b.name]
	})
	return tasks
}

// taskNames extracts the display names from a slice of taskInfo for use in dropdowns.
func taskNames(tasks []taskInfo) []string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.name
	}
	return names
}
