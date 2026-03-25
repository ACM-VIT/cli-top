package tests

import (
	appcmd "cli-top/cmd"
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
		{name: "subcommand help flag", args: []string{"marks", "--help"}, want: true},
		{name: "normal command", args: []string{"marks", "--semester", "1"}, want: false},
		{name: "subcommand value that looks like version", args: []string{"marks", "-v"}, want: false},
		{name: "subcommand long value that looks like version", args: []string{"marks", "--version"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := appcmd.ShouldSkipStartupSideEffects(tt.args); got != tt.want {
				t.Fatalf("ShouldSkipStartupSideEffects(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestShouldSkipCommandSideEffects(t *testing.T) {
	if appcmd.ShouldSkipCommandSideEffects(&cobra.Command{Use: "completion"}, false) == false {
		t.Fatal("completion command should skip side effects")
	}

	if appcmd.ShouldSkipCommandSideEffects(&cobra.Command{Use: "__complete"}, false) == false {
		t.Fatal("__complete command should skip side effects")
	}

	if appcmd.ShouldSkipCommandSideEffects(&cobra.Command{Use: "help"}, false) == false {
		t.Fatal("help command should skip side effects")
	}

	if appcmd.ShouldSkipCommandSideEffects(&cobra.Command{Use: "marks"}, false) {
		t.Fatal("regular commands should not skip side effects")
	}

	helpCmd := &cobra.Command{Use: "marks"}
	helpCmd.Flags().Bool("help", false, "")
	if err := helpCmd.Flags().Set("help", "true"); err != nil {
		t.Fatalf("failed to set help flag: %v", err)
	}
	if appcmd.ShouldSkipCommandSideEffects(helpCmd, false) == false {
		t.Fatal("help flag should skip side effects")
	}

	if appcmd.ShouldSkipCommandSideEffects(&cobra.Command{Use: "cli-top"}, true) == false {
		t.Fatal("root version command should skip side effects")
	}

	if appcmd.ShouldSkipCommandSideEffects(&cobra.Command{Use: "cli-top"}, false) {
		t.Fatal("root command without version flag should not skip side effects")
	}
}
