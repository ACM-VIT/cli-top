package login

import (
	"fmt"
	types "vtop-cli/types"
)

func Login(regNo string, password string) types.Cookies {
	vtopTokens := getLoginPage()

	userInfo := types.LogIn{
		Username: regNo,
		Password: password,
	}

	loginCreds := performLogin(userInfo, vtopTokens)
	fmt.Println(loginCreds)

	return vtopTokens
}
