// main.go

package main

import (
	// features "vtop-cli/features"
	"fmt"
	"log"
	"os"

	// "vtop-cli/features"
	login "vtop-cli/login"
	types "vtop-cli/types"

	"github.com/lpernett/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	userInfo := types.LogIn{
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}
	fmt.Println("(Main) User Info", userInfo)

	loginSecrets := login.Login(userInfo.Username, userInfo.Password)
	fmt.Println("(Main) VTOP Cookies", loginSecrets)
	// features.Marks(userInfo.Username, loginSecrets, 0)
}
