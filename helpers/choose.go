package helpers

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"cli-top/internal/contract"
)

// Choose resolves exact IDs before indexes/queries and never guesses an ambiguous match.
func Choose(subject string, choices []contract.Choice, id, query string, index int) (contract.Choice, error) {
	flag := "--" + strings.ReplaceAll(subject, " ", "-") + "-id"
	if id != "" && (query != "" || index != 0) {
		return contract.Choice{}, contract.New("invalid_arguments", "Use either an exact ID or a legacy selector, not both.")
	}
	fail := func(kind, message string) (contract.Choice, error) {
		e := contract.New(kind, message)
		e.RequiredFlags = []string{flag}
		e.Choices = choices
		return contract.Choice{}, e
	}
	if id != "" {
		for _, c := range choices {
			if c.ID == id {
				return c, nil
			}
		}
		return fail("invalid_selection", "The selected "+subject+" ID is not available in this scope.")
	}
	if index == 0 && query != "" {
		if n, e := strconv.Atoi(query); e == nil {
			if n <= 0 {
				return fail("invalid_selection", "The "+subject+" index is out of range.")
			}
			index = n
			query = ""
		}
	}
	if index != 0 {
		if index > 0 && index <= len(choices) {
			return choices[index-1], nil
		}
		return fail("invalid_selection", "The "+subject+" index is out of range.")
	}
	if query != "" {
		exact := []contract.Choice{}
		matches := []contract.Choice{}
		for _, c := range choices {
			if strings.EqualFold(c.Label, query) {
				exact = append(exact, c)
			}
			if strings.Contains(strings.ToLower(c.Label), strings.ToLower(query)) {
				matches = append(matches, c)
			}
		}
		if len(exact) == 1 {
			return exact[0], nil
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 0 {
			choices = matches
		} else {
			return fail("invalid_selection", "No "+subject+" matches the query.")
		}
	}
	if len(choices) == 1 {
		return choices[0], nil
	}
	if !CanPrompt() || len(choices) == 0 {
		return fail("selection_required", "Select a "+subject+" using "+flag+".")
	}
	for i, c := range choices {
		fmt.Fprintf(OutputWriter(), "%d. %s\n", i+1, c.Label)
	}
	fmt.Fprintf(OutputWriter(), "Choose a %s (number or ID, 'exit' to cancel): ", subject)
	line, err := bufio.NewReader(InputReader()).ReadString('\n')
	if err != nil && line == "" {
		return fail("selection_required", "No selection was provided.")
	}
	line = strings.TrimSpace(line)
	if line == "exit" {
		return contract.Choice{}, contract.New("canceled", "Selection canceled.")
	}
	if n, e := strconv.Atoi(line); e == nil {
		if n <= 0 {
			return fail("invalid_selection", "The "+subject+" index is out of range.")
		}
		return Choose(subject, choices, "", "", n)
	}
	return Choose(subject, choices, line, "", 0)
}
