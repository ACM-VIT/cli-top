package main

import (
	"cli-top/cmd"
	"os"
)

func main() { os.Exit(cmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
