package helpers

import (
	"cli-top/types"
	"regexp"
	"sort"
	"strings"
)

var (
	facultyERPIDRegex = regexp.MustCompile(`^\d+\s*[─–—-]\s*`)
	splitNameRegex    = regexp.MustCompile(`\s*[─–—-]\s*`)
)

func RedactERPID(facultyName string) string {
	return facultyERPIDRegex.ReplaceAllString(facultyName, "")
}

func SplitCourseNameFull(courseName string) []string {
	parts := splitNameRegex.Split(courseName, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func SplitFacultyNameFull(facultyName string) []string {
	parts := splitNameRegex.Split(facultyName, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func ReplaceCrossWithPlus(input string) string {
	return strings.ReplaceAll(input, "┼", "+")
}

func RemoveDuplicateFaculties(faculties []types.FacultyOld) []types.FacultyOld {
	uniqueFaculties := make([]types.FacultyOld, 0, len(faculties))
	keys := make(map[string]bool)
	for _, faculty := range faculties {
		key := faculty.ID + "_" + faculty.Name + "_" + faculty.Slot
		if _, exists := keys[key]; !exists {
			keys[key] = true
			uniqueFaculties = append(uniqueFaculties, faculty)
		}
	}
	return uniqueFaculties
}

func SortFacultiesAlphabetically(faculties []types.FacultyOld) {
	sort.Slice(faculties, func(i, j int) bool {
		return strings.ToLower(faculties[i].Name) < strings.ToLower(faculties[j].Name)
	})
}
