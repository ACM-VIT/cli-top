package cmd


import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var username  = "k"
var password = "k"
var regno ="k"

var credCmd = &cobra.Command{
    Use:   "login",
    Short: "VTOP username, Reg.no and password to be entered",
    Run:   func(cmd *cobra.Command, args []string) {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		regno, _ := cmd.Flags().GetString("regno")

		viper.Set("USERNAME", username)
		viper.Set("PASSWORD", password)
		viper.Set("REGNO", regno)

		if err := viper.WriteConfigAs(".env"); err != nil {
			fmt.Println("Error writing to .env file:", err)
			return
		}

		fmt.Println("Username and password stored in .env file successfully.")
	},
}

func init(){
	
	credCmd.Flags().StringP("username", "u", "", "Enter VTOP username")
	credCmd.Flags().StringP("password", "p", "", "Enter VTOP password")
	credCmd.Flags().StringP("regno", "r", "", "Enter VIT registration number")
	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.ReadInConfig()
	rootCmd.AddCommand(credCmd)

	

   

}

