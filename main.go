package main

import (
	// features "vtop-cli/features"
	"fmt"
	// "vtop-cli/features"
	//"gopkg.in/yaml.v3"
	"gomodules/attendancecalculator"
	//test "gomodules/test"
	types "gomodules/types"
)

func main() {
	cookies := types.Cookies{
		SERVERID:   "s2",
		CSRF:       "b5cc5f45-9372-43ac-8e07-559df2f4f941",
		JSESSIONID: "C932917D52B590F2B0C0208EA482E080",
	}

	// Creating a LogIn instance
	login := types.LogIn{
		Username: "22BCI0272",
		Password: "Hemlo@1005",
	}

	attendancecalculator.PrintSemDetails(login.Username, cookies)

	//semId := attendancecalculator.Marks(login.Username, cookies, 0)
	attendancecalculator.GetAttendance(login.Username, cookies, "")
	fmt.Println("")

}
