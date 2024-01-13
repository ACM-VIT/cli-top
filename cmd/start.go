package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func startfn(cmd *cobra.Command, args []string) {

	red := color.New(color.FgRed)
	blue := color.New(color.FgBlue)
	filePath := "logo.txt"

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	contentStr := string(content)
	ctlen := len(contentStr)
	mid := (ctlen / 2)

	fhlf := contentStr[:mid]
	sndhlf := contentStr[mid:]

	red.Print(fhlf)
	blue.Println(sndhlf)
	red.Println("Welcome to VTOP-CLI(CLITOP)!")
	red.Println("refer to help by typing --help for help or --list for available commands")
}

var rootCmd = &cobra.Command{
	Use:   "vtop-cli",
	Short: "A simple CLI tool for vtop",

	Run: startfn,
}

func Execute() {
	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
