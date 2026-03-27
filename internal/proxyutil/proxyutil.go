package proxyutil

import (
	"fmt"
	"strings"

	"cli-top/helpers"
)

func ParseFlags(args []string) map[string]string {
	if len(args) == 0 {
		return nil
	}
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name := canonicalFlagName(strings.TrimLeft(arg, "-"))
		if name == "" {
			continue
		}
		value := "true"
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			value = args[i+1]
			i++
		}
		flags[name] = value
	}
	if len(flags) == 0 {
		return nil
	}
	return flags
}

func BuildStructuredData(tableSnapshots []helpers.TableSnapshot, selectionRequests []helpers.ProxySelectionRequest, messages []string) map[string]any {
	if len(tableSnapshots) == 0 && len(selectionRequests) == 0 && len(messages) == 0 {
		return nil
	}

	structured := map[string]any{}
	if len(tableSnapshots) > 0 {
		structured["tables"] = tableSnapshots
	}
	if len(selectionRequests) > 0 {
		structured["selection_requests"] = selectionRequests
	}
	if len(messages) > 0 {
		structured["messages"] = messages
	}
	return structured
}

func SelectionRequiredMessage(selectionRequests []helpers.ProxySelectionRequest) string {
	if len(selectionRequests) == 0 {
		return "selection required"
	}
	first := selectionRequests[0]
	if first.Message != "" {
		return first.Message
	}
	if first.Subject != "" {
		return fmt.Sprintf("selection required for %s", first.Subject)
	}
	return "selection required"
}

func NormalizeOutput(raw string) (string, []string) {
	cleaned := strings.TrimSpace(helpers.StripAnsiCodes(raw))
	if cleaned == "" {
		return "", nil
	}

	lines := strings.Split(strings.ReplaceAll(cleaned, "\r\n", "\n"), "\n")
	messages := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		messages = append(messages, trimmed)
	}

	if len(messages) == 0 {
		return "", nil
	}

	return strings.Join(messages, "\n"), messages
}

func canonicalFlagName(flag string) string {
	normalized := strings.ToLower(strings.TrimSpace(flag))
	switch normalized {
	case "s", "semester", "semesterflag", "semesterquery":
		return "semester"
	case "c", "course", "course-name", "coursename":
		return "course"
	case "category", "curriculum-category", "curriculumcategory":
		return "category"
	case "f", "faculty", "facultyflag":
		return "faculty"
	case "facility", "facility-name", "facilityname":
		return "facility"
	case "g", "class-group", "classgroup":
		return "classGroup"
	case "i", "fuzzy-index", "fuzzyindex":
		return "fuzzyIndex"
	case "d", "debug":
		return "debug"
	case "x", "commands", "sync", "sync-commands":
		return "commands"
	case "apply":
		return "apply"
	case "details", "reason":
		return "details"
	case "materials", "material", "topics", "topic-selection":
		return "materials"
	case "assignment", "assignments", "da-selection":
		return "assignment"
	case "confirm", "yes":
		return "confirm"
	case "from-date", "fromdate", "latehourfromdate":
		return "from-date"
	case "from-time", "fromtime":
		return "from-time"
	case "to-date", "todate", "latehourtodate":
		return "to-date"
	case "to-time", "totime":
		return "to-time"
	case "cost-centre-id", "costcentreid", "cost-center-id", "costcenterid":
		return "cost-centre-id"
	case "applied-to", "appliedto", "warden-id", "approver-id":
		return "applied-to"
	case "room-type-id", "roomtypeid":
		return "room-type-id"
	case "event-id", "eventid", "late-hour-event-id", "latehoureventid":
		return "event-id"
	default:
		return normalized
	}
}
