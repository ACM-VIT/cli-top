// main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mycli",
	Short: "A CLI tool to interact with a website",
	Run: func(cmd *cobra.Command, args []string) {
		// Run your default command or display help
		cmd.Help()
	},
}

// main.go (continued)

var websiteCmd = &cobra.Command{
	Use:   "website",
	Short: "Fetch data from a website",
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		fetchWebsiteData(url)
	},
}

func init() {
	// Add flags to the website command
	websiteCmd.Flags().StringP("url", "u", "", "URL of the website")

	// Attach the website command to the root command
	rootCmd.AddCommand(websiteCmd)
}

// Add your website data fetching logic here
func fetchWebsiteData(url string) {
	// Implement logic to fetch and display website data
	fmt.Printf("Fetching data from: %s\n", url)
	// ...
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
