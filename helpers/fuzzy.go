package helpers

import (
	"strings"
	"unicode"
)

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

// FuzzyMatchWithAcronym performs both regular fuzzy matching and acronym matching
// It returns a score: 3 for exact acronym match, 2 for partial acronym match, 1 for fuzzy match, 0 for no match
func FuzzyMatchWithAcronym(query, target string) int {
	query = strings.ToUpper(query)
	target = strings.ToLower(target)

	// Check for empty query
	if len(query) == 0 {
		return 1
	}

	// Check for exact acronym match first (highest priority)
	if exactAcronymMatch(query, target) {
		return 3
	}

	// Check for partial acronym match next (medium priority)
	if acronymMatch(query, target) {
		return 2
	}

	// Fall back to regular fuzzy match (lowest priority)
	// Only do fuzzy match for queries with 3+ characters to avoid too many matches
	if len(query) >= 3 && regularFuzzyMatch(strings.ToLower(query), target) {
		return 1
	}

	return 0
}

func FuzzySearchWithAcronym(nestedList [][]string, stringFlag string) []int {
	var exactAcronymMatches []int
	var partialAcronymMatches []int
	var fuzzyMatches []int

	// For empty query, return all rows except header
	if stringFlag == "" {
		var allRows []int
		for i := 1; i < len(nestedList); i++ {
			allRows = append(allRows, i)
		}
		return allRows
	}

	for i, v := range nestedList {
		// Skip the header row
		if i == 0 {
			continue
		}

		// First check course title and code separately with higher priority
		if len(v) >= 2 {
			courseTitle := v[0]
			courseCode := v[1]

			// Try exact acronym match on title and code first
			if exactAcronymMatch(stringFlag, courseTitle) || exactAcronymMatch(stringFlag, courseCode) {
				exactAcronymMatches = append(exactAcronymMatches, i)
				continue
			}

			// Then try partial acronym match on title and code
			if acronymMatch(stringFlag, courseTitle) || acronymMatch(stringFlag, courseCode) {
				partialAcronymMatches = append(partialAcronymMatches, i)
				continue
			}
		}

		// Fallback: check the entire row
		combinedData := strings.Join(v, " ")
		matchResult := FuzzyMatchWithAcronym(stringFlag, combinedData)
		if matchResult == 3 {
			exactAcronymMatches = append(exactAcronymMatches, i)
		} else if matchResult == 2 {
			partialAcronymMatches = append(partialAcronymMatches, i)
		} else if matchResult == 1 {
			fuzzyMatches = append(fuzzyMatches, i)
		}
	}

	// If any exact acronym matches were found, return them exclusively.
	if len(exactAcronymMatches) > 0 {
		return exactAcronymMatches
	}

	// Otherwise combine partial and fuzzy matches.
	result := append(partialAcronymMatches, fuzzyMatches...)
	return result
}

// computeAcronyms builds two variants: one removing common filler words and one that does not.
func computeAcronyms(target string) (filtered, unfiltered string) {
	words := strings.Fields(target)
	fillerWords := map[string]bool{
		"and":  true,
		"of":   true,
		"for":  true,
		"the":  true,
		"in":   true,
		"with": true,
		"to":   true,
		"a":    true,
		"an":   true,
	}
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		letter := strings.ToUpper(string(word[0]))
		unfiltered += letter // always add first letter
		if !fillerWords[strings.ToLower(word)] {
			filtered += letter
		}
	}
	return
}

// exactAcronymMatch now returns true if the query exactly matches either the filtered or unfiltered acronym.
func exactAcronymMatch(query, target string) bool {
	filtered, unfiltered := computeAcronyms(target)
	return strings.EqualFold(filtered, query) || strings.EqualFold(unfiltered, query)
}

// acronymMatch checks if query matches the first letters of words in target
// using both the filtered and unfiltered acronyms.
func acronymMatch(query, target string) bool {
	words := strings.Fields(target)
	fillerWords := map[string]bool{
		"and":  true,
		"of":   true,
		"for":  true,
		"the":  true,
		"in":   true,
		"with": true,
		"to":   true,
		"a":    true,
		"an":   true,
	}

	// Build four variants:
	var simpleFiltered, compoundFiltered string
	var simpleUnfiltered, compoundUnfiltered string

	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		letter := strings.ToUpper(string(word[0]))
		// Unfiltered
		simpleUnfiltered += letter
		compoundUnfiltered += letter
		for i := 1; i < len(word); i++ {
			if unicode.IsUpper(rune(word[i])) && !unicode.IsUpper(rune(word[i-1])) {
				compoundUnfiltered += string(word[i])
			}
		}
		// Filtered: only add if not filler
		if !fillerWords[strings.ToLower(word)] {
			simpleFiltered += letter
			compoundFiltered += letter
			for i := 1; i < len(word); i++ {
				if unicode.IsUpper(rune(word[i])) && !unicode.IsUpper(rune(word[i-1])) {
					compoundFiltered += string(word[i])
				}
			}
		}
	}

	qUpper := strings.ToUpper(query)
	// Check if query is a substring (or for 2+ chars, a prefix) of any variant
	return strings.Contains(simpleFiltered, qUpper) ||
		strings.Contains(compoundFiltered, qUpper) ||
		strings.Contains(simpleUnfiltered, qUpper) ||
		strings.Contains(compoundUnfiltered, qUpper) ||
		(len(query) >= 2 && (strings.HasPrefix(simpleFiltered, qUpper) ||
			strings.HasPrefix(compoundFiltered, qUpper) ||
			strings.HasPrefix(simpleUnfiltered, qUpper) ||
			strings.HasPrefix(compoundUnfiltered, qUpper)))
}

// regularFuzzyMatch is the core fuzzy matching algorithm
func regularFuzzyMatch(query, target string) bool {
	qLen := len(query)
	tLen := len(target)

	if qLen == 0 {
		return true
	}

	if qLen > tLen {
		return false
	}

	// Check if the query is a substring of the target (highest priority for regular fuzzy)
	if strings.Contains(target, query) {
		return true
	}

	// Check if the query matches the beginning of words in the target
	words := strings.Fields(target)
	for _, word := range words {
		if len(word) >= len(query) && strings.HasPrefix(strings.ToLower(word), query) {
			return true
		}
	}

	// Traditional fuzzy match as fallback
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
