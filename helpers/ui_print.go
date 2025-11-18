package helpers

import "fmt"

// Print ensures the animated headline is cleared before writing to stdout.
func Print(args ...any) {
	StopHeadlineForOutput()
	fmt.Print(args...)
}

// Println ensures the animated headline is cleared before writing a line to stdout.
func Println(args ...any) {
	StopHeadlineForOutput()
	fmt.Println(args...)
}

// Printf ensures the animated headline is cleared before writing formatted output to stdout.
func Printf(format string, args ...any) {
	StopHeadlineForOutput()
	fmt.Printf(format, args...)
}
