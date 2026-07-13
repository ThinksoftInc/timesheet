package main

import (
	"context"
	"fmt"
)

func createTimesheet(ctx context.Context, projectID, taskID, accountID int, date string, unitAmount float64, description string) error {
	if dbg != nil {
		dbg.Printf("createTimesheet: project=%d task=%d account=%d date=%s amount=%.2f desc=%q",
			projectID, taskID, accountID, date, unitAmount, description)
	}
	conn, err := NewConn(ctx)
	if err != nil {
		return fmt.Errorf("connecting to odoo: %w", err)
	}
	id, err := conn.Create(ctx, "account.analytic.line", map[string]any{
		"name":        description,
		"date":        date,
		"unit_amount": unitAmount,
		"account_id":  accountID,
		"project_id":  projectID,
		"task_id":     taskID,
	})
	if err != nil {
		if dbg != nil {
			dbg.Printf("createTimesheet: failed: %v", err)
		}
		return fmt.Errorf("creating timesheet entry: %w", err)
	}
	if dbg != nil {
		dbg.Printf("createTimesheet: created id=%d", id)
	}
	return nil
}
