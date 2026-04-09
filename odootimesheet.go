package main

import (
	"fmt"
)

func createTimesheet(projectID, taskID, accountID int, date string, unitAmount float64, description string) error {
	conn, err := NewConn()
	if err != nil {
		return fmt.Errorf("connecting to odoo: %w", err)
	}
	_, err = conn.Create("account.analytic.line", map[string]any{
		"name":        description,
		"date":        date,
		"unit_amount": unitAmount,
		"account_id":  accountID,
		"project_id":  projectID,
		"task_id":     taskID,
	})
	if err != nil {
		return fmt.Errorf("creating timesheet entry: %w", err)
	}
	return nil
}
