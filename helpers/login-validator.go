package helpers

import (
	types "cli-top/types"
	"fmt"
)

// ValidateLogin checks if the user is logged in by validating the cookies
// Returns true if the user is logged in, otherwise prints a message and returns false
func ValidateLogin(cookies types.Cookies) bool {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login first using the cli-top login command")
		return false
	}
	return true
}
