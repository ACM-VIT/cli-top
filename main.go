// main.go

package main

import (
	// features "vtop-cli/features"
	"fmt"
	login "vtop-cli/login"
)

func main() {
	// Call Marks function with an integer
	secrets := login.Login("2XYYYZZZZ", "password")
	fmt.Println("(Main) VTOP Cookies", secrets)
	// features.Marks(secrets, 0)

}
