package cmd

import (
	"cli-top/debug"
	"cli-top/helpers"
	"fmt"
	"os"
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
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		if strings.TrimSpace(username) == "" {
			username = promptInput("Enter your username: ")
		}
		if password == "" {
			helpers.Println("NOTE: Your password will be visible.")
			password = promptInput("Enter your password: ")
		}
		username = strings.ToUpper(strings.TrimSpace(username))
		if username == "" || password == "" {
			helpers.Println("Username and password are required.")
			return
		}

		key, err := GenerateAESKey()
		if err != nil {
			if debug.Debug {
				helpers.Println(err)
			} else {
				helpers.Println("Unable to initialize secure credential storage.")
			}
			return
		}

		encryptedPassword, err := encryptPassword(password, key)
		if err != nil {
			if debug.Debug {
				helpers.Println("Error encrypting password:", err)
			} else {
				helpers.Println("Unable to securely store the password.")
			}
			return
		}

		helpers.Printf("Logging in with username: %s\n", username)
		viper.Set("VTOP_USERNAME", "\""+username+"\"")
		viper.Set("PASSWORD", "\""+encryptedPassword+"\"")
		viper.Set("KEY", "\""+key+"\"")

		configPath := exeConfigPath()
		if err := viper.WriteConfigAs(configPath); err != nil {
			if debug.Debug {
				helpers.Println("Error writing to .env file:", err)
			} else {
				helpers.Println("Unable to write the cli-top configuration.")
			}
			return
		}
		if err := os.Chmod(configPath, 0o600); err != nil && debug.Debug {
			helpers.Println("Error securing config file permissions:", err)
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
	viper.SetConfigType("env")
	viper.SetConfigFile(exeConfigPath())
	viper.ReadInConfig()
	rootCmd.AddCommand(credCmd)
}
