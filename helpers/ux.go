package helpers

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var processStart = time.Now()

const (
	shimmerPadding       = 10
	shimmerSweepSeconds  = 2.0
	shimmerBandHalf      = 5.0
	shimmerFrameInterval = 45 * time.Millisecond
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiNormal = "\x1b[22m"
	ansiClear  = "\x1b[2K"
)

var (
	activeHeadlineMu   sync.Mutex
	activeHeadlineStop func()

	headlineNewlineMu    sync.Mutex
	headlineNeedsNewline bool
)

func ShouldMuteUI() bool {
	return os.Getenv("CLI_TOP_PROXY_MODE") == "1"
}

func CommandRunner(label string, run func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		muted := ShouldMuteUI()
		start := time.Now()
		defer cleanupHeadlineAnimation()
		if !muted {
			startOrPrintHeadline(label)
		}

		run(cmd, args)

		if !muted {
			stopActiveHeadlineAnimation()
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
	StopHeadlineForOutput()
	fmt.Printf(format, args...)
}

func startOrPrintHeadline(label string) {
	if stopHeadline := startHeadlineAnimation(label); stopHeadline != nil {
		setActiveHeadlineAnimation(stopHeadline)
		return
	}
	headline := renderHeadline(label)
	fmt.Fprintf(color.Output, "\n%s\n", headline)
}

func renderHeadline(label string) string {
	return renderHeadlineWithSupport(label, terminalSupportsColor())
}

func renderHeadlineWithSupport(label string, supportsColor bool) string {
	text := fmt.Sprintf("→ %s", strings.ToUpper(label))
	if !supportsColor {
		return text
	}
	return shimmerLine(text)
}

func shimmerLine(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	pos := shimmerPosition(len(runes))
	var builder strings.Builder
	builder.Grow(len(runes) * 8)
	builder.WriteString(ansiNormal)
	for idx, r := range runes {
		intensity := shimmerIntensity(idx, pos)
		builder.WriteString(ansiForIntensity(intensity))
		builder.WriteRune(r)
	}
	builder.WriteString(ansiReset)
	return builder.String()
}

func shimmerPosition(runeCount int) float64 {
	elapsed := time.Since(processStart).Seconds()
	period := float64(runeCount + shimmerPadding*2)
	phase := math.Mod(elapsed, shimmerSweepSeconds) / shimmerSweepSeconds
	return phase * period
}

func shimmerIntensity(idx int, pos float64) float64 {
	location := float64(idx+shimmerPadding) - pos
	dist := math.Abs(location)
	if dist > shimmerBandHalf {
		return 0
	}
	x := math.Pi * (dist / shimmerBandHalf)
	return 0.5 * (1 + math.Cos(x))
}

func ansiForIntensity(intensity float64) string {
	switch {
	case intensity < 0.2:
		return ansiDim
	case intensity < 0.6:
		return ansiNormal
	default:
		return ansiBold
	}
}

func terminalSupportsColor() bool {
	if color.NoColor {
		return false
	}
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" {
		return false
	}
	return isTerminal()
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func startHeadlineAnimation(label string) func() {
	if ShouldMuteUI() {
		return nil
	}
	supports := terminalSupportsColor()
	if !supports {
		return nil
	}

	initial := renderHeadlineWithSupport(label, supports)
	clearHeadlineNewlineFlag()
	fmt.Fprintf(color.Output, "\n\r%s%s", ansiClear, initial)

	done := make(chan struct{})
	stopped := make(chan struct{})
	var once sync.Once
	ticker := time.NewTicker(shimmerFrameInterval)

	go func() {
		defer close(stopped)
		for {
			select {
			case <-ticker.C:
				fmt.Fprintf(color.Output, "\r%s%s", ansiClear, renderHeadlineWithSupport(label, supports))
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
			<-stopped
			markHeadlineNeedsNewline()
			flushHeadlineNewline()
		})
	}
}

func setActiveHeadlineAnimation(stop func()) {
	activeHeadlineMu.Lock()
	activeHeadlineStop = stop
	activeHeadlineMu.Unlock()
}

func stopActiveHeadlineAnimation() {
	activeHeadlineMu.Lock()
	stop := activeHeadlineStop
	activeHeadlineStop = nil
	activeHeadlineMu.Unlock()
	if stop != nil {
		stop()
	}
}

func StopHeadlineForOutput() {
	stopActiveHeadlineAnimation()
}

func cleanupHeadlineAnimation() {
	stopActiveHeadlineAnimation()
}

func markHeadlineNeedsNewline() {
	headlineNewlineMu.Lock()
	headlineNeedsNewline = true
	headlineNewlineMu.Unlock()
}

func clearHeadlineNewlineFlag() {
	headlineNewlineMu.Lock()
	headlineNeedsNewline = false
	headlineNewlineMu.Unlock()
}

func flushHeadlineNewline() {
	headlineNewlineMu.Lock()
	needed := headlineNeedsNewline
	if needed {
		headlineNeedsNewline = false
	}
	headlineNewlineMu.Unlock()
	if needed {
		fmt.Fprint(color.Output, "\n")
	}
}
