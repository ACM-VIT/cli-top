package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd := &cobra.Command{
		Use:   "help",
		Short: "Help page for vtop-cli",
		Long:  "VTOP CLI HELP TEXT GOES HERE BRRRR.",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	// Command to handle help
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "help [command]",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				rootCmd.Help()
				return
			}
			for _, arg := range args {
				found, _, _ := rootCmd.Find(args)
				if found == nil {
					fmt.Printf("Unknown help topic %#q\n", arg)
					os.Exit(1)
				}
				found.Help()
			}
		},
	})

	
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println("VTOP HELP TEXT")
	})

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-h" || arg == "--help" {
				rootCmd.Help()
				os.Exit(0)
			}
		}
	}
}
