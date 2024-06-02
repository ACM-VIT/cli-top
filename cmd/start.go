package cmd

import (
	"fmt"

	// "io/ioutil"
	"cli-top/debug"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/login"
	"cli-top/types"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/lpernett/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cookies types.Cookies
var userInfo types.LogIn
var semesterFlag int
var debugFlag bool
var versionFlag bool
var updateFlag bool

func startfn(cmd *cobra.Command, args []string) {

	red := color.New(color.FgRed)
	blue := color.New(color.FgBlue)
	// filePath := "logo.txt"

	// content, err := ioutil.ReadFile(filePath)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// contentStr := string(content)
	contentStr := logo()
	ctlen := len(contentStr)
	mid := (ctlen / 2)

	fhlf := contentStr[:mid]
	sndhlf := contentStr[mid:]

	red.Print(fhlf)
	blue.Println(sndhlf)
	red.Println("Welcome to CLI-TOP!\n ")
	red.Println("Use \"cli-top help\" or \"cli-top --list\" to show available commands\nUse \"cli-top [command] --help\" for more information about a command.\n ")
	fileName := "cli-top-config.env"

	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	// Construct the full path to the file
	filePath := filepath.Join(currentDir, fileName)

	// Check if the file exists
	if _, err := os.Stat(filePath); err == nil {
		if debug.Debug {
			fmt.Println("File exists:", filePath)
		}

		err := godotenv.Load("cli-top-config.env")
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		if debug.Debug {
			fmt.Println(os.Getenv("PASSWORD"))
		}

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

func vtop_login() (types.Cookies, string) {
	err := godotenv.Load("cli-top-config.env")
	if err != nil {
		log.Fatal("Error loading .env file, please enter your credentials using the \"login\" command.")
	}

	userInfo := types.LogIn{
		Username: os.Getenv("VTOP_USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}

	key := os.Getenv("KEY")

	password, err := decryptPassword(userInfo.Password, key)
	if err != nil {
		fmt.Println("Error decrypting password:", err)
	}

	loginSecrets := login.Login(userInfo.Username, password)
	cookies, tmp := login.HomePage(loginSecrets)
	userInfo.RegNo = tmp

	saveCookiesToFile(cookies, userInfo, key)
	if err != nil {
		fmt.Println("Error saving cookies:", err)
	}
	if debug.Debug {
		fmt.Println("(Main) VTOP Cookies", cookies)
	}

	return cookies, userInfo.RegNo
}

func saveCookiesToFile(cookies types.Cookies, userInfo types.LogIn, Key string) {
	viper.Set("CSRF", "\""+cookies.CSRF+"\"")
	viper.Set("JSESSIONID", "\""+cookies.JSESSIONID+"\"")
	viper.Set("SERVERID", "\""+cookies.SERVERID+"\"")
	viper.Set("REGNO", "\""+userInfo.RegNo+"\"")
	viper.Set("VTOP_USERNAME", "\""+userInfo.Username+"\"")
	viper.Set("PASSWORD", "\""+userInfo.Password+"\"")
	viper.Set("KEY", "\""+Key+"\"")
	if err := viper.WriteConfigAs("cli-top-config.env"); err != nil {
		fmt.Println("Error writing to .env file:", err)
	}
}

func readCookiesFromFile() (types.Cookies, string) {
	if debugFlag {
		debug.Debug = true
		fmt.Println("Debug mode on")
	}
	err := godotenv.Load("cli-top-config.env")
	if err != nil {
		log.Fatal("Error loading .env file, please enter your credentials using the \"login\" command.")
	}
	cookies := types.Cookies{
		SERVERID:   os.Getenv("SERVERID"),
		CSRF:       os.Getenv("CSRF"),
		JSESSIONID: os.Getenv("JSESSIONID"),
	}
	cookies, regNo := login.HomePage(cookies)
	if regNo == "" {
		cookies, regNo = vtop_login()
	}

	return cookies, regNo
}

var rootCmd = &cobra.Command{
	Use:   "cli-top",
	Short: "A simple CLI tool for vtop",

	Run: func(cmd *cobra.Command, args []string) {

		if debugFlag {
			debug.Debug = true
			fmt.Println("Debug mode on")
		}

		if versionFlag {
			fmt.Println("Version:", debug.Version)
			return
		}

		if updateFlag {
			helpers.CheckUpdate()
			return
		}

		startfn(cmd, args)
	},
}

func Execute() {
	killSwitch := helpers.CheckKillSwitch()
	if killSwitch == 2 {
		fmt.Println("This version of cli-top has been decomissioned. Please await an update at https://cli-top.acmvit.in/.")
		return
		// os.Exit(1)
	}
	rootCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Print Debug Messages")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print Version Number")
	rootCmd.PersistentFlags().BoolVarP(&updateFlag, "update", "u", false, "Check for Updates")
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(marksCmd)
	rootCmd.AddCommand(gradesCmd)
	rootCmd.AddCommand(attendanceCmd)
	rootCmd.AddCommand(timeTableCmd)
	rootCmd.AddCommand(receiptCmd)
	rootCmd.AddCommand(hostelCmd)
	rootCmd.AddCommand(cgpaCmd)
	rootCmd.AddCommand(examScheduleCmd)
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
		features.Marks(regNo, cookies, semesterFlag)
	},
}

var gradesCmd = &cobra.Command{
	Use:   "grades",
	Short: "Show Grade Details of a particular semester",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetGrades(regNo, cookies, "", semesterFlag)
	},
}

var attendanceCmd = &cobra.Command{
	Use:   "attendance",
	Short: "Show Attendance Details of a particular semester",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetAttendance(regNo, cookies, "", semesterFlag)
	},
}

var receiptCmd = &cobra.Command{
	Use:   "receipts",
	Short: "Show Receipt Details of a user",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetReceipt(regNo, cookies)
	},
}

var timeTableCmd = &cobra.Command{
	Use:   "timetable",
	Short: "Show Time Table of a particular semester",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetTimeTable(regNo, cookies, "", semesterFlag)
	},
}

var hostelCmd = &cobra.Command{
	Use:   "hostel",
	Short: "Show Hostel Details of a user",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintHostelInfo(regNo, cookies, "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView")
	},
}

var cgpaCmd = &cobra.Command{
	Use:   "cgpa",
	Short: "Show CGPA details",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintCgpa(regNo, cookies, "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory")
	},
}

var examScheduleCmd = &cobra.Command{
	Use:   "exams",
	Short: "Show Exam Schedule",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetExamSchedule(regNo, cookies, "", semesterFlag)
	},
}
