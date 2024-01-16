package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

var username = "k"
var password = "k"
var regno = "k"

var credCmd = &cobra.Command{
	Use:   "login",
	Short: "VTOP username and password to be entered",
	Run: func(cmd *cobra.Command, args []string) {
		username := promptInput("Enter your username: ")
		password := promptPassword("Enter your password: ")
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

		if err := viper.WriteConfigAs(".env"); err != nil {
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

func promptPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("\nError reading password:", err)
		os.Exit(1)
	}
	fmt.Println() // Print a new line after password input
	password := string(bytePassword)
	return password
}

func init() {
	credCmd.Flags().StringP("username", "u", "", "Enter VTOP username")
	credCmd.Flags().StringP("password", "p", "", "Enter VTOP password")
	credCmd.Flags().StringP("regno", "r", "", "Enter VIT registration number")
	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.ReadInConfig()
	rootCmd.AddCommand(credCmd)
}
