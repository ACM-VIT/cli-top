package debug

import (
	"fmt"
	"time"
)

var Debug bool = false
var Version string = "2.9.8"

func Log(message string) {
	if Debug {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[DEBUG %s] %s\n", timestamp, message)
	}
}
