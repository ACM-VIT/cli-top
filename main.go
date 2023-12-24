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
		CSRF:       "7f6e6f77-c3d2-482b-bdb2-f74bdcf97356",
		JSESSIONID: "C9F77ED32FE5E018A74B6428FD67AA05",
	}

	// Creating a LogIn instance
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

