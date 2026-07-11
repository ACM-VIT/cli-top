package helpers

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
)

type headlineCapture struct {
	mu         sync.Mutex
	output     strings.Builder
	redrawn    chan struct{}
	redrawOnce sync.Once
}

func newHeadlineCapture() *headlineCapture {
	return &headlineCapture{redrawn: make(chan struct{})}
}

func (capture *headlineCapture) Write(data []byte) (int, error) {
	capture.mu.Lock()
	_, _ = capture.output.Write(data)
	capture.mu.Unlock()
	if bytes.Contains(data, []byte(ansiUp)) {
		capture.redrawOnce.Do(func() { close(capture.redrawn) })
	}
	return len(data), nil
}

func (capture *headlineCapture) String() string {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.output.String()
}

func TestHeadlineShimmerRemainsEnabledForColorTerminals(t *testing.T) {
	rendered := renderHeadlineWithSupport("marks", true)
	if !strings.Contains(rendered, "\x1b[") {
		t.Fatalf("color headline has no shimmer formatting: %q", rendered)
	}
	if got := StripAnsiCodes(rendered); got != "→ MARKS" {
		t.Fatalf("rendered headline text = %q", got)
	}
}

func TestHeadlineFallsBackToPlainText(t *testing.T) {
	if got := renderHeadlineWithSupport("marks", false); got != "→ MARKS" {
		t.Fatalf("plain headline = %q", got)
	}
}

func TestHeadlineAnimationRedrawsAndStopsOnColorTerminals(t *testing.T) {
	capture := newHeadlineCapture()
	previousOutput := color.Output
	color.Output = capture
	t.Cleanup(func() { color.Output = previousOutput })

	stop := startHeadlineAnimationWithSupport("marks", true)
	if stop == nil {
		t.Fatal("color terminal did not start the headline animation")
	}

	select {
	case <-capture.redrawn:
	case <-time.After(time.Second):
		stop()
		t.Fatal("headline animation did not redraw")
	}

	stop()
	stop()
	output := capture.String()
	if !strings.Contains(output, ansiUp) {
		t.Fatalf("animation output has no cursor redraw: %q", output)
	}
	if !strings.Contains(output, ansiBold+"→ MARKS"+ansiReset) {
		t.Fatalf("animation output has no completed headline: %q", output)
	}
}
