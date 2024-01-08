// main.go

package main

import (
	// features "vtop-cli/features"
	"fmt"
	"log"
	"os"

	// "vtop-cli/features"

	"vtop-cli/features"
	"vtop-cli/login"
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
		Username: os.Getenv("VTOP_USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}
	RegNo := os.Getenv("REGNO")
	fmt.Println("(Main) User Info", userInfo, RegNo)

	loginSecrets := login.Login(userInfo.Username, userInfo.Password)
	cookies := login.HomePage(loginSecrets)
	fmt.Println("(Main) VTOP Cookies", cookies)
	features.Profile(cookies, RegNo)

	// ac.PrintSemDetails(RegNo, cookies)
	// ac.GetAttendance(RegNo, cookies, "")
	// fmt.Println("")
}
