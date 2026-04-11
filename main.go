package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// dateOptions returns dates from today back to the 1st of 2 months ago,
// with today as the default (index 0).
func dateOptions() ([]string, int) {
	today := time.Now()
	first := time.Date(today.Year(), today.Month()-2, 1, 0, 0, 0, 0, today.Location())
	var dates []string
	for d := today; !d.Before(first); d = d.AddDate(0, 0, -1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates, 0
}

// durationOptions returns time durations from 00:30 to 24:00 in 30-minute steps.
func durationOptions() []string {
	opts := make([]string, 0, 48)
	for m := 30; m <= 24*60; m += 30 {
		opts = append(opts, fmt.Sprintf("%02d:%02d", m/60, m%60))
	}
	return opts
}

// parseDuration converts a "HH:MM" string to fractional hours.
// e.g. "00:30" → 0.5, "01:30" → 1.5, "02:00" → 2.0
func parseDuration(s string) float64 {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	hoursStr := strings.TrimSpace(parts[0])
	minsStr := strings.TrimSpace(parts[1])
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		return 0
	}
	minutes, err := strconv.Atoi(minsStr)
	if err != nil {
		return 0
	}
	// Validate sensible ranges: non-negative hours, minutes in [0,59].
	if hours < 0 || minutes < 0 || minutes >= 60 {
		return 0
	}
	return float64(hours) + float64(minutes)/60.0
}

// formatDescription builds the timesheet entry name from the selected GitHub
// issue and the text area content.
//   - blank text area → "/"
//   - no issue selected → text (or "/")
//   - issue selected → "#123 - text"
func formatDescription(issueText, areaText string) string {
	text := strings.TrimSpace(areaText)
	if text == "" {
		text = "/"
	}
	if issueText == "" {
		return text
	}
	// Issue options are formatted "#123 Some title" — extract just the number.
	num := 0
	if strings.HasPrefix(issueText, "#") {
		end := strings.Index(issueText, " ")
		numStr := issueText[1:]
		if end > 0 {
			numStr = issueText[1:end]
		}
		if n, err := strconv.Atoi(numStr); err == nil {
			num = n
		}
	}
	if num == 0 {
		return text
	}
	return fmt.Sprintf("#%d - %s", num, text)
}

func main() {
	if _, err := loadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprintf(os.Stderr, "Create %s with the following contents:\n\n", configFilePath())
		fmt.Fprint(os.Stderr, "[connection]\n")
		fmt.Fprint(os.Stderr, "hostname = \"odoo.example.com\"\n")
		fmt.Fprint(os.Stderr, "port     = 8069\n")
		fmt.Fprint(os.Stderr, "schema   = \"http\"\n")
		fmt.Fprint(os.Stderr, "database = \"mycompany\"\n")
		fmt.Fprint(os.Stderr, "username = \"user@example.com\"\n")
		fmt.Fprint(os.Stderr, "apikey   = \"your-api-key\"\n")
		os.Exit(1)
	}

	// Color palette — colorblind-safe, terminal-consistent.
	const (
		bgApp       = tcell.Color19            // #0000af — app background, between navy and medium blue
		bgDropdown  = tcell.ColorNavy          // #000080 — dropdown list non-selected (darker, creates depth)
		bgTextArea  = tcell.ColorDarkSlateGray // text area background — dark but distinct from navy
		fgTextArea  = tcell.ColorWhite         // text on dark text area bg
		colBlue     = tcell.ColorBlue          // save button normal
		colBlueDk   = tcell.Color18            // #000087 — save button activated (dark blue)
		colOrange   = tcell.ColorOrange        // quit button / error state
		colOrangeDk = tcell.Color130           // #af5f00 — quit activated / error activated
	)

	// Reusable styles — bold labels and buttons.
	lblStyle := tcell.StyleDefault.Foreground(tcell.ColorSilver).Bold(true)
	saveBtnStyle := tcell.StyleDefault.Background(colBlue).Foreground(tcell.ColorWhite).Bold(true)
	saveBtnAct := tcell.StyleDefault.Background(colBlueDk).Foreground(tcell.ColorWhite).Bold(true)
	saveBtnErr := tcell.StyleDefault.Background(colOrange).Foreground(tcell.ColorWhite).Bold(true)
	quitBtnStyle := tcell.StyleDefault.Background(colOrange).Foreground(tcell.ColorBlack).Bold(true)
	quitBtnAct := tcell.StyleDefault.Background(colOrangeDk).Foreground(tcell.ColorBlack).Bold(true)

	// Apply global theme before NewApplication so all primitives inherit it.
	tview.Styles.PrimitiveBackgroundColor = bgApp
	tview.Styles.ContrastBackgroundColor = tcell.ColorMidnightBlue
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorDarkBlue
	tview.Styles.BorderColor = tcell.ColorLightSlateGray
	tview.Styles.TitleColor = tcell.ColorWhite
	tview.Styles.GraphicsColor = tcell.ColorLightSlateGray
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorSilver
	tview.Styles.TertiaryTextColor = tcell.ColorGray
	tview.Styles.InverseTextColor = tcell.ColorBlack
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorYellow

	app := tview.NewApplication()

	// Use a cancellable context for background loads so we can cancel inflight
	// Odoo requests when the user makes a new selection or exits the app.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	projectInfos := projects(rootCtx)
	names := projectNames(projectInfos)
	initialProject := ""
	if len(names) > 0 {
		initialProject = names[0]
	}

	// Track the currently selected project and its tasks so the save button
	// can pass IDs directly to createTimesheet without extra Odoo lookups.
	var selectedProject *projectInfo
	if len(projectInfos) > 0 {
		selectedProject = &projectInfos[0]
	}
	currentTasks := projectTasks(rootCtx, initialProject)
	selectedTaskIdx := 0
	taskChangedFn := func(_ string, idx int) {
		selectedTaskIdx = idx
	}

	// Resolve initial git URL and issues from the first project's description.
	var initialIssues []string
	initialGitViewText := " Repo: "
	if selectedProject != nil {
		initialGitURL := parseGitURL(selectedProject.description)
		initialGitViewText = " Repo: " + initialGitURL
		var ghErr error
		// Use rootCtx so the initial fetch is cancellable when the app exits.
		initialIssues, ghErr = githubIssues(rootCtx, initialGitURL)
		if ghErr != nil {
			initialGitViewText += "  (" + ghErr.Error() + ")"
		}
	}

	// Create dropdowns and the git URL view before the form so the project
	// callback can reference them.
	taskDropdown := tview.NewDropDown().
		SetLabel("Task").
		SetOptions(taskNames(currentTasks), taskChangedFn)

	gitURLView := tview.NewTextView().
		SetText(initialGitViewText)

	issueDropdown := tview.NewDropDown().
		SetLabel("GH Issue").
		SetOptions(initialIssues, nil)

	dates, todayIdx := dateOptions()
	durations := durationOptions()

	// Track selected date and duration for the save button.
	selectedDate := dates[todayIdx]
	selectedDuration := parseDuration(durations[0])

	// Calculate project dropdown width from the longest project name.
	// Account for the label "Project: " prefix and dropdown decoration.
	maxNameLen := 0
	for _, n := range names {
		if l := utf8.RuneCountInString(n); l > maxNameLen {
			maxNameLen = l
		}
	}
	projectWidth := maxNameLen + len("Project: ") + 6 // padding for dropdown chrome

	// loadGen is incremented on every project selection. Each spawned goroutine
	// captures its generation at dispatch time and discards its result if a newer
	// selection has since been made, preventing stale data from overwriting the UI.
	var loadGen int
	var loadCancel context.CancelFunc

	// Create standalone forms for the side-by-side row.
	projectForm := tview.NewForm().
		AddDropDown("Project", names, 0, func(option string, _ int) {
			loadGen++
			gen := loadGen
			// Cancel any previous in-flight load and start a new one with a
			// bounded timeout so slow network calls don't hang the UI.
			if loadCancel != nil {
				loadCancel()
			}
			loadCtx, cancel := context.WithTimeout(rootCtx, 8*time.Second)
			loadCancel = cancel

			go func(gen int, opt string, ctx context.Context) {
				tasks := projectTasks(ctx, opt)
				var gitViewText string
				var issues []string
				p := findProject(projectInfos, opt)
				if p != nil {
					gitURL := parseGitURL(p.description)
					gitViewText = " Repo: " + gitURL
					var ghErr error
					// Use the same load context so issue fetches are cancelled when
					// the project selection changes.
					issues, ghErr = githubIssues(ctx, gitURL)
					if ghErr != nil {
						gitViewText += "  (" + ghErr.Error() + ")"
					}
				}
				app.QueueUpdateDraw(func() {
					if gen != loadGen {
						return // stale — a newer selection is in flight
					}
					selectedProject = p
					currentTasks = tasks
					selectedTaskIdx = 0
					taskDropdown.SetOptions(taskNames(tasks), taskChangedFn)
					taskDropdown.SetCurrentOption(0)
					gitURLView.SetText(gitViewText)
					issueDropdown.SetOptions(issues, nil)
					issueDropdown.SetCurrentOption(0)
				})
			}(gen, option, loadCtx)
		})

	taskForm := tview.NewForm().
		AddFormItem(taskDropdown)

	// Horizontal row: project (fixed width) | task (fill remaining).
	topRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(projectForm, projectWidth, 0, true).
		AddItem(taskForm, 0, 1, false)

	// Issue form (standalone, repo is display-only and not in any form).
	issueForm := tview.NewForm().
		AddFormItem(issueDropdown)

	// Date / Duration side-by-side row.
	dateForm := tview.NewForm().
		AddDropDown("Date", dates, todayIdx, func(option string, _ int) {
			selectedDate = option
		})

	durationForm := tview.NewForm().
		AddDropDown("Duration", durations, 0, func(option string, _ int) {
			selectedDuration = parseDuration(option)
		})

	dateRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(dateForm, 24, 0, false).
		AddItem(durationForm, 0, 1, false)

	// Create the TextArea separately so we can intercept Tab/Backtab on it
	// (TextArea normally captures Tab as text input).
	timesheetArea := tview.NewTextArea().
		SetLabel("Timesheet ")
	timesheetArea.SetBorder(true)
	timesheetArea.SetBorderPadding(0, 0, 1, 4)
	timesheetArea.SetBackgroundColor(bgTextArea)
	timesheetArea.SetTextStyle(tcell.StyleDefault.Background(bgTextArea).Foreground(fgTextArea))
	timesheetArea.SetLabelStyle(lblStyle)

	// Button bar: individual buttons so we can style them independently.
	var saveBtn *tview.Button
	var saving bool
	saveBtn = tview.NewButton("Save").SetSelectedFunc(func() {
		if saving {
			return
		}
		// Capture current selection in the UI goroutine before spawning.
		p := selectedProject
		tasks := currentTasks
		taskIdx := selectedTaskIdx
		date := selectedDate
		unitAmount := selectedDuration
		_, issueText := issueDropdown.GetCurrentOption()
		description := formatDescription(issueText, timesheetArea.GetText())
		saving = true
		go func() {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			result := make(chan error, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						result <- fmt.Errorf("panic: %v", r)
					}
				}()
				var projectID, taskID, accountID int
				if p != nil {
					projectID = p.id
					accountID = p.accountID
				}
				if taskIdx >= 0 && taskIdx < len(tasks) {
					taskID = tasks[taskIdx].id
				}
				// Tie the save operation to rootCtx with a timeout so the
				// request can be cancelled if the app shuts down.
				saveCtx, saveCancel := context.WithTimeout(rootCtx, 30*time.Second)
				defer saveCancel()
				result <- createTimesheet(saveCtx, projectID, taskID, accountID, date, unitAmount, description)
			}()
			ticker := time.NewTicker(80 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					frame := frames[i%len(frames)]
					i++
					app.QueueUpdateDraw(func() {
						saveBtn.SetLabel(frame)
					})
				case err := <-result:
					if err != nil {
						app.QueueUpdateDraw(func() {
							saveBtn.SetLabel("Error!")
							saveBtn.SetStyle(saveBtnErr)
						})
						time.Sleep(2000 * time.Millisecond)
						app.QueueUpdateDraw(func() {
							saveBtn.SetLabel("Save")
							saveBtn.SetStyle(saveBtnStyle)
							saving = false
						})
					} else {
						app.QueueUpdateDraw(func() {
							saveBtn.SetLabel("Saved!")
						})
						time.Sleep(1500 * time.Millisecond)
						app.QueueUpdateDraw(func() {
							saveBtn.SetLabel("Save")
							saving = false
							// Reset per-entry fields; project/task stay for follow-up entries.
							dateForm.GetFormItemByLabel("Date").(*tview.DropDown).SetCurrentOption(todayIdx)
							selectedDate = dates[todayIdx]
							durationForm.GetFormItemByLabel("Duration").(*tview.DropDown).SetCurrentOption(0)
							selectedDuration = parseDuration(durations[0])
							issueDropdown.SetCurrentOption(-1)
							timesheetArea.SetText("", true)
							app.SetFocus(projectForm)
						})
					}
					return
				}
			}
		}()
	})
	saveBtn.SetStyle(saveBtnStyle)
	saveBtn.SetActivatedStyle(saveBtnAct)

	quitBtn := tview.NewButton("Quit").SetSelectedFunc(func() {
		app.Stop()
	})
	quitBtn.SetStyle(quitBtnStyle)
	quitBtn.SetActivatedStyle(quitBtnAct)

	// Center buttons with spacers on each side and a gap between.
	buttonBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(saveBtn, 8, 0, false).
		AddItem(tview.NewBox(), 2, 0, false).
		AddItem(quitBtn, 8, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	timesheetArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(saveBtn)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(durationForm)
			return nil
		}
		return event
	})

	// Wire up tab progression across all forms in order:
	// projectForm -> taskForm -> issueForm -> dateForm -> durationForm -> timesheetArea -> saveBtn -> quitBtn -> (wrap)

	projectForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(taskForm)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(quitBtn)
			return nil
		}
		return event
	})

	taskForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(issueForm)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(projectForm)
			return nil
		}
		return event
	})

	issueForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(dateForm)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(taskForm)
			return nil
		}
		return event
	})

	dateForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(durationForm)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(issueForm)
			return nil
		}
		return event
	})

	durationForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Reset to the timesheet area when entering.
			app.SetFocus(timesheetArea)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(dateForm)
			return nil
		}
		return event
	})

	saveBtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(quitBtn)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(timesheetArea)
			return nil
		}
		return event
	})

	quitBtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(projectForm)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(saveBtn)
			return nil
		}
		return event
	})

	// Apply consistent list styles to all dropdowns: non-selected is the
	// colour-inverted form of the selected item (navy bg / white text ↔ white bg / navy text).
	ddUnselected := tcell.StyleDefault.Background(bgDropdown).Foreground(tcell.ColorWhite)
	ddSelected := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(bgDropdown)
	applyDD := func(dd *tview.DropDown) {
		dd.SetListStyles(ddUnselected, ddSelected)
		dd.SetLabelStyle(lblStyle)
	}
	applyDD(taskDropdown)
	applyDD(issueDropdown)
	applyDD(projectForm.GetFormItemByLabel("Project").(*tview.DropDown))
	applyDD(dateForm.GetFormItemByLabel("Date").(*tview.DropDown))
	applyDD(durationForm.GetFormItemByLabel("Duration").(*tview.DropDown))

	// Stack: top row, repo display, issue form, date row, timesheet, buttons.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topRow, 3, 0, true).
		AddItem(gitURLView, 1, 0, false).
		AddItem(issueForm, 3, 0, false).
		AddItem(dateRow, 3, 0, false).
		AddItem(timesheetArea, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(buttonBar, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false)

	// Outer box provides the screen border.
	outer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(layout, 0, 1, true)
	outer.SetBorder(true).SetTitle("Timesheet Entry").SetTitleAlign(tview.AlignCenter)

	if err := app.SetRoot(outer, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
