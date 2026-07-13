package main

import (
	"log"
	"os"
	"path/filepath"
)

// dbg is the package-level debug logger. It is nil unless the -log flag is set.
var dbg *log.Logger

// initLogger opens (or creates) the log file at ~/.local/share/thinksoft/timesheet.log
// and sets dbg to a logger writing only to the file (never stderr/stdout).
func initLogger() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".local", "share", "thinksoft")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	logPath := filepath.Join(dir, "timesheet.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	dbg = log.New(f, "[timesheet] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	return nil
}

// boolStr returns "yes" or "no" for use in log messages.
func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
