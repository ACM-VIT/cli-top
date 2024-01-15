package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"vtop-cli/features"
	"vtop-cli/login"
	"vtop-cli/types"

	"github.com/fatih/color"
	"github.com/lpernett/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cookies types.Cookies
var userInfo types.LogIn

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
	red.Println("Welcome to VTOP-CLI!")
	red.Println("refer to help by typing --help for help or --list for available commands")
	fileName := ".env"

	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	// Construct the full path to the file
	filePath = filepath.Join(currentDir, fileName)
	fmt.Println(filePath)

	// Check if the file exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Println("File exists:", filePath)
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		fmt.Println(os.Getenv("PASSWORD"))
		if os.Getenv("VTOP_USERNAME") != "" && os.Getenv("PASSWORD") != "" {
			vtop_login()
		}
	} else if os.IsNotExist(err) {
		fmt.Println("File does not exist:", filePath)
		fmt.Println("Please login using the \"login\" command")
	} else {
		fmt.Println("Error checking file existence:", err)
	}
}

func vtop_login() types.Cookies {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	userInfo := types.LogIn{
		Username: os.Getenv("VTOP_USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}

	loginSecrets := login.Login(userInfo.Username, userInfo.Password)
	cookies, tmp := login.HomePage(loginSecrets)
	userInfo.RegNo = tmp
	fmt.Println(userInfo.RegNo)
	saveCookiesToFile(cookies, userInfo)
	if err != nil {
		fmt.Println("Error saving cookies:", err)
	}
	fmt.Println("(Main) VTOP Cookies", cookies)
	return cookies
}

func saveCookiesToFile(cookies types.Cookies, userInfo types.LogIn) {
	viper.Set("CSRF", "\""+cookies.CSRF+"\"")
	viper.Set("JSESSIONID", "\""+cookies.JSESSIONID+"\"")
	viper.Set("SERVERID", "\""+cookies.SERVERID+"\"")
	viper.Set("REGNO", "\""+userInfo.RegNo+"\"")
	viper.Set("VTOP_USERNAME", "\""+userInfo.Username+"\"")
	viper.Set("PASSWORD", "\""+userInfo.Password+"\"")
	if err := viper.WriteConfigAs(".env"); err != nil {
		fmt.Println("Error writing to .env file:", err)
	}
	return
}

func readCookiesFromFile() (types.Cookies, string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	cookies := types.Cookies{
		SERVERID:   os.Getenv("SERVERID"),
		CSRF:       os.Getenv("CSRF"),
		JSESSIONID: os.Getenv("JSESSIONID"),
	}
	cookies, regno := login.HomePage(cookies)
	if regno == "" {
		cookies = vtop_login()
	}
	regNo := os.Getenv("REGNO")
	return cookies, regNo
}

var rootCmd = &cobra.Command{
	Use:   "vtop-cli",
	Short: "A simple CLI tool for vtop",

	Run: startfn,
}

func Execute() {
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(marksCmd)
	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show VTOP Student Profile",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.Profile(cookies, regNo)
	},
}

var marksCmd = &cobra.Command{
	Use:   "marks",
	Short: "Show Marks Details of a particular semester",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.Marks(regNo, cookies, 0)
	},
}
