package login

import (
	"fmt"
	types "vtop-cli/types"
)

func Login(regNo string, password string) types.Cookies {
	vtopTokens, captcha := getLoginPage()
	fmt.Println(captcha)

	userInfo := types.LogIn{
		Username: regNo,
		Password: password,
	}

	loginCreds := performLogin(userInfo, vtopTokens, captcha)
	fmt.Println(loginCreds)

	return vtopTokens
}
