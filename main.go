// main.go

package main

import (
	// features "vtop-cli/features"
	"fmt"
	// "vtop-cli/features"
	login "vtop-cli/login"
	types "vtop-cli/types"
)

func main() {
	// Call Marks function with an integer
	userInfo := types.LogIn{
		Username: "2XYYYZZZZ",
		Password: "password",
	}
	fmt.Println(userInfo)
	// loginSecrets := login.Login(userInfo.Username, userInfo.Password)
	// fmt.Println("(Main) VTOP Cookies", loginSecrets)
	// features.Marks(userInfo.Username, loginSecrets, 0)

	fmt.Print(login.CaptchaParse("output_image.jpg"))

}
