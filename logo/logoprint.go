package logo

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

func PrintLogo() {
	// Set up color printers
	red := color.New(color.FgRed)
	blue := color.New(color.FgBlue)

	currentDir, _ := os.Getwd()
	fmt.Println("Current Working Directory:", currentDir)

	// Read the contents of the text file
	content, err := os.ReadFile("logo.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Convert the content to a string
	text := string(content)

	// Get the length of the content
	ctlen := len(text)

	// Calculate the midpoint
	mid := ctlen / 2

	// Iterate through each character in the string
	for i := 0; i < ctlen; i++ {
		// Apply the provided coloring logic
		if i <= mid {
			red.Print(string(text[i]))
		} else {
			blue.Print(string(text[i]))
		}

		// Print a newline after reaching the midpoint

	}
}
