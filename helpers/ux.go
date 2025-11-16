package helpers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func ShouldMuteUI() bool {
	return os.Getenv("CLI_TOP_PROXY_MODE") == "1"
}

func CommandRunner(label string, run func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		muted := ShouldMuteUI()
		start := time.Now()
		if !muted {
			headline := fmt.Sprintf("→ %s", strings.ToUpper(label))
			color.New(color.FgHiCyan).Printf("\n%s\n", headline)
		}

		run(cmd, args)

		if !muted {
			elapsed := time.Since(start).Round(10 * time.Millisecond)
			successLine := fmt.Sprintf("✓ %s (%s)", strings.ToUpper(label), elapsed)
			color.New(color.FgHiGreen).Printf("\n%s\n\n", successLine)
		}
	}
}

func Infof(format string, args ...any) {
	if ShouldMuteUI() {
		return
	}
	fmt.Printf(format, args...)
}
