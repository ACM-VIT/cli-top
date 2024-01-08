
package cmd


import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Store username and password",
	Run: func(cmd *cobra.Command, args []string) {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		if username == "" || password == "" {
			fmt.Println("Please provide both username and password")
			return
		}

		saveCredentials(username, password)
	},
}

func saveCredentials(username, password string) {
	viper.Set("username", username)
	viper.Set("password", password)

	err := viper.WriteConfigAs("credentials.yml")
	if err != nil {
		fmt.Println("Error saving credentials:", err)
		return
	}
	fmt.Println("Credentials saved successfully!")
}

func Execute() {
	rootCmd.Flags().StringP("username", "u", "", "Enter VTOP username")
	rootCmd.Flags().StringP("password", "p", "", "Enter VTOP password")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("No existing login data found, creating a new one.")
		} else {
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
