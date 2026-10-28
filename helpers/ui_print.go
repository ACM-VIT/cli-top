package helpers

import "fmt"

// Print ensures the animated headline is cleared before writing to stdout.
func Print(args ...any) {
	StopHeadlineForOutput()
	fmt.Fprint(OutputWriter(), args...)
}

// Println ensures the animated headline is cleared before writing a line to stdout.
func Println(args ...any) {
	StopHeadlineForOutput()
	fmt.Fprintln(OutputWriter(), args...)
}

// Printf ensures the animated headline is cleared before writing formatted output to stdout.
func Printf(format string, args ...any) {
	StopHeadlineForOutput()
	fmt.Fprintf(OutputWriter(), format, args...)
}
