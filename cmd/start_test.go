package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestShouldSkipStartupSideEffects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "completion command", args: []string{"completion", "zsh"}, want: true},
		{name: "hidden complete command", args: []string{"__complete", "marks"}, want: true},
		{name: "help flag", args: []string{"--help"}, want: true},
		{name: "version flag", args: []string{"-v"}, want: true},
		{name: "normal command", args: []string{"marks", "--semester", "1"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldSkipStartupSideEffects(tt.args); got != tt.want {
				t.Fatalf("shouldSkipStartupSideEffects(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestShouldSkipCommandSideEffects(t *testing.T) {
	originalVersionFlag := versionFlag
	t.Cleanup(func() {
		versionFlag = originalVersionFlag
	})

	versionFlag = false
	if shouldSkipCommandSideEffects(&cobra.Command{Use: "completion"}) == false {
		t.Fatal("completion command should skip side effects")
	}

	if shouldSkipCommandSideEffects(&cobra.Command{Use: "__complete"}) == false {
		t.Fatal("__complete command should skip side effects")
	}

	if shouldSkipCommandSideEffects(&cobra.Command{Use: "help"}) == false {
		t.Fatal("help command should skip side effects")
	}

	if shouldSkipCommandSideEffects(&cobra.Command{Use: "marks"}) {
		t.Fatal("regular commands should not skip side effects")
	}

	helpCmd := &cobra.Command{Use: "marks"}
	helpCmd.Flags().Bool("help", false, "")
	if err := helpCmd.Flags().Set("help", "true"); err != nil {
		t.Fatalf("failed to set help flag: %v", err)
	}
	if shouldSkipCommandSideEffects(helpCmd) == false {
		t.Fatal("help flag should skip side effects")
	}

	versionFlag = true
	if shouldSkipCommandSideEffects(&cobra.Command{Use: "cli-top"}) == false {
		t.Fatal("root version command should skip side effects")
	}

	versionFlag = false
	if shouldSkipCommandSideEffects(&cobra.Command{Use: "cli-top"}) {
		t.Fatal("root command without version flag should not skip side effects")
	}
}
