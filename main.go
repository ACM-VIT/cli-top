package main

import (
	// features "vtop-cli/features"
	"fmt"
	// "vtop-cli/features"
	//"gopkg.in/yaml.v3"
	// "VTOP-CLI/attendancecalculator"
	//test "gomodules/test"
	types "VTOP-CLI/types"
	"VTOP-CLI/gradeView"
)

func main() {
	cookies := types.Cookies{
		SERVERID:   "s1",
		CSRF:       "9bc8e552-952d-4eca-a0e7-5dfc05c44f4e",
		JSESSIONID: "4219A0A1DF403087A8D6EC3BA45BB640",
	}

	
	login := types.LogIn{
		Username: "22BCI0272",
		Password: "",
	}

	// attendancecalculator.PrintSemDetails(login.Username, cookies)
	// attendancecalculator.GetAttendance(login.Username, cookies, "")
	gradeView.PrintSemDetails(login.Username,cookies)
	gradeView.GetGrade(login.Username,cookies,"")
	fmt.Println("")

}

