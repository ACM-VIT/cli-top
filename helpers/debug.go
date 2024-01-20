package helpers

import "fmt"

var Debug bool = false

func SetDebug() {
	Debug = true
	if Debug {
		fmt.Print("Debug Mode is on")
	}
}
