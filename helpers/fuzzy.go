package helpers

import "strings"

func FuzzyMatch(query, target string) bool {
	query = strings.ToLower(query)
	target = strings.ToLower(target)

	qLen := len(query)
	tLen := len(target)

	if qLen == 0 {
		return true
	}
	if qLen > tLen {
		return false
	}

	q := 0
	for i := 0; i < tLen; i++ {
		if query[q] == target[i] {
			q++
			if q == qLen {
				return true
			}
		}
	}
	return false
}
