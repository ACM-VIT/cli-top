package login

import types "vtop-cli/types"

func Login(regNo string, password string) string {
	secrets := getLoginPage()

	vtopTokens := types.Tokens{
		AuthID:     regNo,
		Csrf:       secrets.CSRF,
		JsessionID: secrets.JSESSIONID,
		ServerID:   secrets.SERVERID,
	}

	userInfo := types.LogIn{
		Username: regNo,
		Password: password,
	}

	loginCreds := performLogin(userInfo, vtopTokens)

	return loginCreds
}
