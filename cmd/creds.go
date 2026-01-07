package cmd

import (
	"cli-top/debug"
	"cli-top/helpers"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func exeConfigPath() string {
	return helpers.ConfigFilePath()
}

var credCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to VTOP",
	Run: func(cmd *cobra.Command, args []string) {
		helpers.Println("NOTE: Your password will be visible.")
		username := promptInput("Enter your username: ")
		password := promptInput("Enter your password: ")
		key := GenerateAESKey()

		encryptedPassword, err := encryptPassword(password, key)
		if err != nil && debug.Debug {
			helpers.Println("Error encrypting password:", err)
			return
		}

		helpers.Printf("Logging in with username: %s\n", strings.ToUpper(username))
		viper.Set("VTOP_USERNAME", "\""+strings.ToUpper(username)+"\"")
		viper.Set("PASSWORD", "\""+encryptedPassword+"\"")
		viper.Set("KEY", "\""+key+"\"")

		if err := viper.WriteConfigAs(exeConfigPath()); err != nil && debug.Debug {
			helpers.Println("Error writing to .env file:", err)
			return
		}

		helpers.Println("Username and encrypted password stored in .env file successfully.")
	},
}

func promptInput(prompt string) string {
	helpers.Print(prompt)
	var input string
	fmt.Scanln(&input)
	return input
}

func init() {
	credCmd.Flags().String("username", "", "Enter VTOP username")
	credCmd.Flags().String("password", "", "Enter VTOP password")
	credCmd.Flags().String("regno", "", "Enter VIT registration number")
	viper.SetConfigType("env")
	viper.SetConfigFile(exeConfigPath())
	viper.ReadInConfig()
	rootCmd.AddCommand(credCmd)
}
