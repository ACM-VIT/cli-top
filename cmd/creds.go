package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var username = "k"
var password = "k"
var regno = "k"

var credCmd = &cobra.Command{
	Use:   "login",
	Short: "VTOP username and password to be entered",
	Run: func(cmd *cobra.Command, args []string) {
		username := promptInput("Enter your username: ")
		password := promptInput("Enter your password: ")
		key := GenerateAESKey()

		encryptedPassword, err := encryptPassword(password, key)
		if err != nil {
			fmt.Println("Error encrypting password:", err)
			return
		}

		fmt.Printf("Logging in with username: %s\n", strings.ToUpper(username))
		viper.Set("VTOP_USERNAME", "\""+strings.ToUpper(username)+"\"")
		viper.Set("PASSWORD", "\""+encryptedPassword+"\"")
		viper.Set("KEY", "\""+key+"\"")

		if err := viper.WriteConfigAs("vtop.env"); err != nil {
			fmt.Println("Error writing to .env file:", err)
			return
		}

		fmt.Println("Username and encrypted password stored in .env file successfully.")
	},
}

func promptInput(prompt string) string {
	fmt.Print(prompt)
	var input string
	fmt.Scanln(&input)
	return input
}

func init() {
	credCmd.Flags().StringP("username", "u", "", "Enter VTOP username")
	credCmd.Flags().StringP("password", "p", "", "Enter VTOP password")
	credCmd.Flags().StringP("regno", "r", "", "Enter VIT registration number")
	viper.SetConfigType("env")
	viper.SetConfigFile("vtop.env")
	viper.ReadInConfig()
	rootCmd.AddCommand(credCmd)
}
