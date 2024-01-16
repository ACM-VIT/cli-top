// main.go

package main

import (
	//features "vtop-cli/features"
	"fmt"
	"os"
	"log"
	// "vtop-cli/features"
	//"vtop-cli/features/schedule-gen"
	"vtop-cli/features"
	"vtop-cli/login"
	types "vtop-cli/types"
	"github.com/lpernett/godotenv"

	"os/exec"
)

func main() {

	cmd := exec.Command("python", "features/schedule-gen.py")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	schedule_gen := string(output)
	

	// Load .env file
	if err = godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

	userInfo := types.LogIn{
		Username: os.Getenv("VTOP_USERNAME"),
		Password: os.Getenv("PASSWORD"),
		RegNo:    "",
	}
	
	//fmt.Println("(Main) User Info", userInfo)

	loginSecrets := login.Login(userInfo.Username, userInfo.Password)
	cookies, tmp := login.HomePage(loginSecrets)
	userInfo.RegNo = tmp
	// fmt.Println("(Main) Registration Number", userInfo.RegNo)
	//fmt.Println("(Main) VTOP Cookies", cookies)
	//features.Profile(cookies, userInfo.RegNo)

	features.PrintSemDetails(userInfo.RegNo,cookies)
	//features.GetGrade(userInfo.RegNo,cookies,"")
	//features.GetAttendance(userInfo.RegNo,cookies,"")

	features.GetTimeTable(userInfo.RegNo,cookies,"",schedule_gen)
	
	
	fmt.Println("")
}
