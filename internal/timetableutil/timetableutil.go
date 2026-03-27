package timetableutil

import "time"

func ShouldPromptSemesterSelection(now time.Time) bool {
	switch now.Month() {
	case time.June, time.July, time.December:
		return true
	default:
		return false
	}
}
