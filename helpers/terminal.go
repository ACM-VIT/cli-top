package helpers

import (
	"os"

	"golang.org/x/term"
)

func IsInteractiveOutput(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}
