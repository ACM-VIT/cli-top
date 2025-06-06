package helpers

import (
	"fmt"
	"strings"
)

// PrintGradientText prints the given text using a diagonal gradient from the provided colors.
// gradColors should be a slice of 3 hex color strings (e.g., "#ff0000").
func PrintGradientText(text string, gradColors []string) {
	lines := strings.Split(text, "\n")
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}
	for y, line := range lines {
		for x, ch := range line {
			t := float64(x+y) / float64(maxLen+len(lines)-2)
			color := getGradientColor(gradColors, t)
			printColoredChar(ch, color)
		}
		fmt.Println()
	}
}

// getGradientColor returns a hex color string interpolated between the gradient colors based on t in [0,1].
func getGradientColor(gradColors []string, t float64) string {
	if len(gradColors) < 3 {
		gradColors = []string{"#ff0000", "#00ff00", "#0000ff"}
	}
	if t <= 0 {
		return gradColors[0]
	}
	if t >= 1 {
		return gradColors[2]
	}
	if t < 0.5 {
		return interpolateHexColor(gradColors[0], gradColors[1], t*2)
	}
	return interpolateHexColor(gradColors[1], gradColors[2], (t-0.5)*2)
}

// interpolateHexColor linearly interpolates between two hex colors.
func interpolateHexColor(a, b string, t float64) string {
	ar, ag, ab := hexToRGB(a)
	br, bg, bb := hexToRGB(b)
	r := uint8(float64(ar) + (float64(br)-float64(ar))*t)
	g := uint8(float64(ag) + (float64(bg)-float64(ag))*t)
	bb2 := uint8(float64(ab) + (float64(bb)-float64(ab))*t)
	return fmt.Sprintf("#%02x%02x%02x", r, g, bb2)
}

// hexToRGB converts a hex color string to RGB values.
func hexToRGB(hex string) (r, g, b uint8) {
	var rr, gg, bb int
	fmt.Sscanf(hex, "#%02x%02x%02x", &rr, &gg, &bb)
	return uint8(rr), uint8(gg), uint8(bb)
}

// printColoredChar prints a single rune with the given hex color using ANSI 24-bit color.
func printColoredChar(ch rune, hex string) {
	r, g, b := hexToRGB(hex)
	fmt.Printf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, string(ch))
}
