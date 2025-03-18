package cmd

import (
	"bytes"
	"cli-top/debug"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/login"
	types "cli-top/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/lpernett/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var semesterFlag int
var debugFlag bool
var versionFlag bool
var updateFlag bool
var courseFlag int
var facultyFlag string
var classGrpFlag int
var fuzzyIndexFlag int
var courseNameFlag string
var syllabusCourseFlag string

func getOrCreateUUID() string {
	registeredUUID := viper.GetString("UUID")
	if registeredUUID != "" {
		return registeredUUID
	}

	unregisteredUUID := viper.GetString("UNREGISTERED_UUID")
	if unregisteredUUID == "" {
		unregisteredUUID = uuid.New().String()
		viper.Set("UNREGISTERED_UUID", unregisteredUUID)
		if err := viper.WriteConfigAs("cli-top-config.env"); err != nil && debug.Debug {
			fmt.Println("Error saving unregistered UUID to config:", err)
		}
	}

	if err := helpers.RegisterUUID(unregisteredUUID); err != nil {
		if debug.Debug {
			fmt.Println("Error registering UUID with server:", err)
		}
		return unregisteredUUID
	}

	viper.Set("UUID", unregisteredUUID)
	viper.Set("UNREGISTERED_UUID", "")
	if err := viper.WriteConfigAs("cli-top-config.env"); err != nil && debug.Debug {
		fmt.Println("Error updating registered UUID in config:", err)
	}

	return unregisteredUUID
}

func trackCommand(command string) {
	userUUID := viper.GetString("UUID")
	if userUUID == "" {
		if debug.Debug {
			log.Println("UUID is empty or not initialized. Skipping tracking.")
		}
		return
	}

	data := types.TrackingData{
		UUID:      userUUID,
		Command:   command,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		if debug.Debug {
			log.Println("Error marshaling tracking data:", err)
		}
		return
	}

	serverURL := "https://cli-calendar.acmvit.in/track"

	req, err := http.NewRequest("POST", serverURL, bytes.NewBuffer(jsonData))
	if err != nil {
		if debug.Debug {
			log.Println("Error creating tracking request:", err)
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", data.UUID)

	client := &http.Client{Timeout: 10 * time.Second}

	// Send the POST request asynchronously
	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			log.Println("Error sending tracking data:", err)
		}
		return
	}
	defer resp.Body.Close()

	// Discard the response body to free resources
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		if debug.Debug {
			log.Println("Invalid UUID detected. Generating a new one and registering...")
		}
		newUUID := uuid.New().String()
		err = helpers.RegisterUUID(newUUID)
		if err != nil {
			if debug.Debug {
				log.Println("Failed to register new UUID:", err)
			}
			return
		}
		viper.Set("UUID", newUUID)
		viper.Set("UNREGISTERED_UUID", "")
		if err := viper.WriteConfigAs("cli-top-config.env"); err != nil && debug.Debug {
			fmt.Println("Error updating registered UUID in config:", err)
		}
	} else if resp.StatusCode != http.StatusOK {
		if debug.Debug {
			log.Println("Unexpected response status during tracking:", resp.Status)
		}
	} else {
		if debug.Debug {
			log.Println("Tracking data sent successfully.")
		}
	}
}

func startfn() {
	red := color.New(color.FgHiRed)
	//blue := color.New(color.FgHiBlue)
	pink := color.New(color.FgHiMagenta)

	contentStr := logo()
	for _, char := range contentStr {
		switch char {
		// Dripping elements (blue)
		case '█', '▀', '▄', '▓':
			pink.Print(string(char))
		// Regular characters (red)
		default:
			pink.Print(string(char))
		}
	}
	red.Println("\nWelcome to CLI-TOP!\n ")
	red.Println("Use \"cli-top help\" or \"cli-top --list\" to show available commands\nUse \"cli-top [command] --help\" for more information about a command.\n ")
	fileName := "cli-top-config.env"

	currentDir, err := os.Getwd()
	if err != nil && debug.Debug {
		fmt.Println("Error getting current directory:", err)
		return
	}

	filePath := filepath.Join(currentDir, fileName)

	if _, err := os.Stat(filePath); err == nil {
		if debug.Debug {
			fmt.Println("File exists:", filePath)
		}
		err := godotenv.Load("cli-top-config.env")
		if err != nil && debug.Debug {
			fmt.Println("Error loading .env file")
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

	userUUID := getOrCreateUUID()
	if debug.Debug {
		fmt.Println("User UUID:", userUUID)
	}
}

func vtop_login() (types.Cookies, string) {
	err := godotenv.Load("cli-top-config.env")
	if err != nil && debug.Debug {
		fmt.Println("Error loading .env file, please enter your credentials using the \"login\" command.")
	}

	userInfo := types.LogIn{
		Username: os.Getenv("VTOP_USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}

	key := os.Getenv("KEY")

	password, err := decryptPassword(userInfo.Password, key)
	if err != nil && debug.Debug {
		fmt.Println("Error decrypting password:", err)
	}

	loginSecrets := login.Login(userInfo.Username, password)
	cookies, tmp := login.HomePage(loginSecrets)
	userInfo.RegNo = tmp

	saveCookiesToFile(cookies, userInfo, key)
	if err != nil && debug.Debug {
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
	if err := viper.WriteConfigAs("cli-top-config.env"); err != nil && debug.Debug {
		fmt.Println("Error writing to .env file:", err)
	}
}

func readCookiesFromFile() (types.Cookies, string) {
	if debugFlag {
		debug.Debug = true
		fmt.Println("Debug mode on")
	}
	err := godotenv.Load("cli-top-config.env")
	if err != nil && debug.Debug {
		fmt.Println("Error loading .env file, please enter your credentials using the \"login\" command.")
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

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() != "login" && cmd.Name() != "logout" && cmd.Name() != "cli-top" {
			go trackCommand(cmd.Name())
		}

		// if cmd.Name() != "login" && cmd.Name() != "logout" && cmd.Name() != "cli-top" {
		// 	go func() {
		// 		userUUID := viper.GetString("UUID")
		// 		if userUUID == "" {
		// 			return
		// 		}

		// 		data := types.VersionTrackingData{
		// 			UUID:      userUUID,
		// 			Command:   cmd.Name(),
		// 			Version:   debug.Version,
		// 			Timestamp: time.Now().Format(time.RFC3339),
		// 		}

		// 		jsonData, err := json.Marshal(data)
		// 		if err != nil {
		// 			if debug.Debug {
		// 				log.Println("Error marshaling version tracking data:", err)
		// 			}
		// 			return
		// 		}

		// 		serverURL := "https://cli-calendar.acmvit.in/version-track"

		// 		req, err := http.NewRequest("POST", serverURL, bytes.NewBuffer(jsonData))
		// 		if err != nil {
		// 			if debug.Debug {
		// 				log.Println("Error creating version tracking request:", err)
		// 			}
		// 			return
		// 		}

		// 		req.Header.Set("Content-Type", "application/json")
		// 		req.Header.Set("x-api-key", userUUID)

		// 		client := &http.Client{Timeout: 10 * time.Second}
		// 		resp, err := client.Do(req)
		// 		if err != nil {
		// 			if debug.Debug {
		// 				log.Println("Error sending version tracking data:", err)
		// 			}
		// 			return
		// 		}
		// 		defer resp.Body.Close()

		// 		io.Copy(io.Discard, resp.Body)
		// 	}()
		// }
	},

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

		startfn()
	},
}

func init() {
	rootCmd.SetUsageTemplate(`Usage:
  {{.CommandPath}} [global flags] <subcommand> [subcommand flags] [arguments]
{{if .HasAvailableLocalFlags}}

Global Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
{{if .HasAvailableSubCommands}}

Available Subcommands:{{range .Commands}}{{if (and .IsAvailableCommand (not .Hidden))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}

Use "{{.CommandPath}} <subcommand> --help" for more information about a subcommand.
`)
}

func Execute() {
	killSwitch := helpers.CheckKillSwitch()
	if killSwitch == 2 {
		fmt.Println("This version of cli-top has been decommissioned. Please await an update at https://cli-top.acmvit.in/.")
		return
	}

	err := godotenv.Load("cli-top-config.env")
	if err != nil && debug.Debug {
		fmt.Println("Error loading .env file:", err)
	}

	userUUID := getOrCreateUUID()
	if debug.Debug {
		fmt.Println("User UUID:", userUUID)
	}

	if !updateFlag && os.Args[len(os.Args)-1] != "-u" && os.Args[len(os.Args)-1] != "--update" {
		shouldNotify, latestVersion := helpers.ShouldShowUpdateNotification()
		if shouldNotify {
			helpers.ShowUpdateNotification(latestVersion)
		}
	}

	// Define flags for subcommands
	marksCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	gradesCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	timeTableCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	examScheduleCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	calendarCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	calendarCmd.PersistentFlags().IntVarP(&classGrpFlag, "class-group", "g", 0, "Specify the class group")
	coursePageCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	coursePageCmd.PersistentFlags().IntVarP(&courseFlag, "course", "c", 0, "Specify the course")
	coursePageCmd.PersistentFlags().StringVarP(&facultyFlag, "faculty", "f", "", "Specify the faculty")
	coursePageCmd.PersistentFlags().IntVarP(&fuzzyIndexFlag, "fuzzy-index", "i", 0, "Specify the fuzzy index")
	syllabusCmd.PersistentFlags().StringVarP(&syllabusCourseFlag, "course", "c", "", "Specify course search query ") 
	//daDetailsCmd.PersistentFlags().StringVarP(&courseNameFlag, "course-name", "c", "", "Specify the course name")
	// Define global flags
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Print Debug Messages")
	rootCmd.PersistentFlags().BoolVarP(&updateFlag, "update", "u", false, "Check for Updates")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print Version Number")

	// Add subcommands to root command
	rootCmd.AddCommand(profileCmd, marksCmd, gradesCmd, attendanceCmd, timeTableCmd, receiptCmd, hostelCmd, cgpaCmd, examScheduleCmd, libraryDuesCmd, logoutCmd, calendarCmd, coursePageCmd, nightslipCmd, leavestatusCmd, classMessagesCmd, daDetailsCmd, facilityCmd, syllabusCmd)

	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil && debug.Debug {
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

var facilityCmd = &cobra.Command{
	Use:   "facility",
	Short: "Register for facilities",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.RegisterPhyFacility(regNo, cookies)
	},
}

var syllabusCmd = &cobra.Command{
	Use:   "syllabus",
	Short: "Download syllabus for a selected course",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.ExecuteSyllabusDownload(regNo, cookies, syllabusCourseFlag)
	},
}

var marksCmd = &cobra.Command{
	Use:   "marks",
	Short: "Show Marks Details of a particular semester",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetMarks(regNo, cookies, "", semesterFlag)
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
		features.GetAttendance(regNo, cookies, semesterFlag)
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
		features.GetExamSchedule(regNo, cookies, semesterFlag)
	},
}

var coursePageCmd = &cobra.Command{
	Use:   "course-page",
	Short: "Download course materials for a selected semester, course, and faculty",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.ExecuteCoursePageDownload(regNo, cookies, semesterFlag, courseFlag, facultyFlag, fuzzyIndexFlag)
	},
}

var libraryDuesCmd = &cobra.Command{
	Use:   "library-dues",
	Short: "Show Library Dues",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetLibraryDues(regNo, cookies)
	},
}

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Show Calendar",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintCal(regNo, cookies, semesterFlag, classGrpFlag)
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from VTOP",
	Run: func(cmd *cobra.Command, args []string) {
		err := godotenv.Load("cli-top-config.env")
		if err != nil && debug.Debug {
			fmt.Println("Error loading .env file:", err)
			return
		}

		uuid := os.Getenv("UUID")

		if uuid == "" {
			fmt.Println("UUID not found; nothing to preserve.")
			return
		}

		env := map[string]string{
			"UUID": uuid,
		}

		f, err := os.Create("cli-top-config.env")
		if err != nil {
			if debug.Debug {
				fmt.Println("Error creating .env file:", err)
			}
			return
		}
		defer f.Close()

		for key, value := range env {
			_, err = f.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			if err != nil && debug.Debug {
				fmt.Println("Error writing to .env file:", err)
				return
			}
		}

		fmt.Println("Logged out successfully.")
	},
}

var nightslipCmd = &cobra.Command{
	Use:   "nightslip",
	Short: "Show Nightslip Request Status of a user",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetNightSlipStatus(regNo, cookies)
	},
}

var leavestatusCmd = &cobra.Command{
	Use:   "leave",
	Short: "Show Leave Status",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetLeaveStatus(regNo, cookies)
	},
}

var classMessagesCmd = &cobra.Command{
	Use:   "msg",
	Short: "Show Class Messages",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetClassMessage(regNo, cookies)
	},
}

var daDetailsCmd = &cobra.Command{
	Use:   "da",
	Short: "Show Digital Assignment Details",
	Run: func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintAllDAs(regNo, cookies, courseNameFlag)
	},
}

var updateCmd = &cobra.Command{
	Use:  "update",
	Short: "Perform updates",
	Run: func(cmd *cobra.Command, args []string) {
		features.Update()
	},
}