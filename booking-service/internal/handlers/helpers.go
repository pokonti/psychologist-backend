package handlers

import "time"

// Helper to parse "2026-01-01"
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// Helper to merge date "2026-01-01" with time "14:30"
func combineDateAndTime(date time.Time, timeStr string) (time.Time, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, err
	}
	// Return new date with specific hour/minute
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC), nil
}
