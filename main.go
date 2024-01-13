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
		RegNo:    "",
	}

	// fmt.Println("(Main) User Info", userInfo)

	loginSecrets := login.Login(userInfo.Username, userInfo.Password)
	cookies, tmp := login.HomePage(loginSecrets)
	userInfo.RegNo = tmp
	// fmt.Println("(Main) Registration Number", userInfo.RegNo)
	fmt.Println("(Main) VTOP Cookies", cookies)
	features.Profile(cookies, userInfo.RegNo)
	features.PrintHostelInfo(userInfo.RegNo, cookies, "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView" )
	features.PrintCgpa(userInfo.RegNo, cookies, "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory")
	// features.PrintRawHostelInfo1(userInfo.RegNo, cookies)
	// features.PrintRawLeaveHistory(userInfo.RegNo, cookies)
	// features.HostelInfo(userInfo.RegNo, cookies)
	// features.CgpaView(userInfo.RegNo, cookies)
	// features.LeaveHistory(userInfo.RegNo, cookies)

	// ac.PrintSemDetails(RegNo, cookies)
	// ac.GetAttendance(RegNo, cookies, "")
	// fmt.Println("")
}
