// main.go

package main

import (
	features "vtop-cli/features"
	login "vtop-cli/login"
)

func main() {
	// Call Marks function with an integer
	secrets := login.Login("21BCI0028")
	features.Marks(secrets, 0)

}
