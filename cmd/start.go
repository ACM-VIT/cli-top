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
	"strings"
	"time"

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

func configFilePath() string {
	return helpers.ConfigFilePath()
}

func getOrCreateUUID() string {
	registeredUUID := viper.GetString("UUID")
	if registeredUUID != "" {
		return registeredUUID
	}

	unregisteredUUID := viper.GetString("UNREGISTERED_UUID")
	if unregisteredUUID == "" {
		unregisteredUUID = uuid.New().String()
		viper.Set("UNREGISTERED_UUID", unregisteredUUID)
		if err := viper.WriteConfigAs(configFilePath()); err != nil && debug.Debug {
			helpers.Println("Error saving unregistered UUID to config:", err)
		}
	}

	go func(uuid string) {
		if err := helpers.RegisterUUID(uuid); err != nil && debug.Debug {
			helpers.Println("Error registering UUID with server:", err)
		}
	}(unregisteredUUID)

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

	serverURL := helpers.CalendarServerURL + "/track"

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
		if err := viper.WriteConfigAs(configFilePath()); err != nil && debug.Debug {
			helpers.Println("Error updating registered UUID in config:", err)
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
	reset := "\x1b[0m"
	grays := []string{
		"\x1b[38;5;250m",
		"\x1b[38;5;248m",
		"\x1b[38;5;245m",
		"\x1b[38;5;243m",
		"\x1b[38;5;240m",
		"\x1b[38;5;238m",
	}
	dim := "\x1b[38;5;102m"
	text := "\x1b[38;5;145m"

	fmt.Println()
	lines := strings.Split(logo(), "\n")
	colorIdx := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			fmt.Println()
			continue
		}
		colorCode := grays[colorIdx%len(grays)]
		colorIdx++
		fmt.Printf("%s%s%s\n", colorCode, line, reset)
	}
	fmt.Printf("\n%sWelcome to CLI-TOP!%s\n\n", text, reset)
	fmt.Printf("%sUse \"cli-top help\" or \"cli-top --list\" to show available commands%s\n", dim, reset)
	fmt.Printf("%sUse \"cli-top [command] --help\" for more information about a command.%s\n\n", dim, reset)
	filePath := configFilePath()

	if _, err := os.Stat(filePath); err == nil {
		if debug.Debug {
			helpers.Println("File exists:", filePath)
		}
		err := godotenv.Load(filePath)
		helpers.LoadSemesterCacheFromEnv()
		if err != nil && debug.Debug {
			helpers.Println("Error loading .env file")
		}
		if debug.Debug {
			helpers.Println(os.Getenv("PASSWORD"))
		}

		if os.Getenv("VTOP_USERNAME") != "" && os.Getenv("PASSWORD") != "" {
			vtop_login()
		}
	} else {
		// File not found in cwd or exe dir
		if debug.Debug {
			helpers.Println("File does not exist:", filePath)
		}
		helpers.Println("Please login using the \"login\" command")
	}

	userUUID := getOrCreateUUID()
	if debug.Debug {
		helpers.Println("User UUID:", userUUID)
	}
}

func vtop_login() (types.Cookies, string) {
	err := godotenv.Load(configFilePath())
	helpers.LoadSemesterCacheFromEnv()
	if err != nil && debug.Debug {
		helpers.Println("Error loading .env file, please enter your credentials using the \"login\" command.")
	}

	userInfo := types.LogIn{
		Username: os.Getenv("VTOP_USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}

	key := os.Getenv("KEY")

	password, err := decryptPassword(userInfo.Password, key)
	if err != nil && debug.Debug {
		helpers.Println("Error decrypting password:", err)
	}

	loginSecrets := login.Login(userInfo.Username, password)
	cookies, tmp := login.HomePage(loginSecrets)
	userInfo.RegNo = tmp

	saveCookiesToFile(cookies, userInfo, key)
	if err != nil && debug.Debug {
		helpers.Println("Error saving cookies:", err)
	}
	if debug.Debug {
		helpers.Println("(Main) VTOP Cookies", cookies)
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
	if err := viper.WriteConfigAs(configFilePath()); err != nil && debug.Debug {
		helpers.Println("Error writing to .env file:", err)
	}
	if userInfo.RegNo != "" {
		helpers.InvalidateSemesterCache(userInfo.RegNo)
	}
}

func readCookiesFromFile() (types.Cookies, string) {
	if debugFlag {
		debug.Debug = true
		helpers.Println("Debug mode on")
	}
	err := godotenv.Load(configFilePath())
	if err != nil && debug.Debug {
		helpers.Println("Error loading .env file, please enter your credentials using the \"login\" command.")
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

func isStartupSideEffectFreeArg(arg string) bool {
	switch arg {
	case "completion", "__complete", "__completeNoDesc", "help", "--help", "-h", "--version", "-v":
		return true
	default:
		return false
	}
}

type startupArgsInfo struct {
	remainingArgs        []string
	rootHelpRequested    bool
	rootVersionRequested bool
	rootUpdateRequested  bool
}

func inspectStartupArgs(args []string) startupArgsInfo {
	info := startupArgsInfo{}

	for i, arg := range args {
		switch {
		case arg == "--":
			if i+1 < len(args) {
				info.remainingArgs = args[i+1:]
			}
			return info
		case arg == "" || arg == "-" || !strings.HasPrefix(arg, "-"):
			info.remainingArgs = args[i:]
			return info
		case strings.HasPrefix(arg, "--"):
			switch arg {
			case "--debug":
				continue
			case "--update":
				info.rootUpdateRequested = true
				continue
			case "--version":
				info.rootVersionRequested = true
				return info
			case "--help":
				info.rootHelpRequested = true
				return info
			default:
				info.remainingArgs = args[i:]
				return info
			}
		default:
			recognizedCluster := true
			for _, shortFlag := range arg[1:] {
				switch shortFlag {
				case 'd':
				case 'u':
					info.rootUpdateRequested = true
				case 'v':
					info.rootVersionRequested = true
					return info
				case 'h':
					info.rootHelpRequested = true
					return info
				default:
					recognizedCluster = false
				}
			}

			if recognizedCluster {
				continue
			}

			info.remainingArgs = args[i:]
			return info
		}
	}

	return info
}

func shouldSkipStartupSideEffects(info startupArgsInfo) bool {
	if info.rootHelpRequested || info.rootVersionRequested {
		return true
	}

	if len(info.remainingArgs) == 0 {
		return false
	}

	if isStartupSideEffectFreeArg(info.remainingArgs[0]) {
		return true
	}

	for _, arg := range info.remainingArgs[1:] {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}

	return false
}

func ShouldSkipStartupSideEffects(args []string) bool {
	return shouldSkipStartupSideEffects(inspectStartupArgs(args))
}

func ShouldSkipCommandSideEffects(cmd *cobra.Command, rootVersionRequested bool) bool {
	if cmd == nil {
		return false
	}

	if flag := cmd.Flags().Lookup("help"); flag != nil && flag.Changed {
		return true
	}
	if flag := cmd.InheritedFlags().Lookup("help"); flag != nil && flag.Changed {
		return true
	}

	switch cmd.Name() {
	case "completion", "__complete", "__completeNoDesc", "help":
		return true
	case "cli-top":
		return rootVersionRequested
	default:
		return false
	}
}

var rootCmd = &cobra.Command{
	Use:   "cli-top",
	Short: "A simple CLI tool for vtop",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if ShouldSkipCommandSideEffects(cmd, versionFlag) {
			return
		}

		if cmd.Name() != "login" && cmd.Name() != "logout" && cmd.Name() != "cli-top" && cmd.Name() != "proxy" {
			go trackCommand(cmd.Name())
		}

		if cmd.Name() != "login" && cmd.Name() != "logout" && cmd.Name() != "proxy" {
			commandName := cmd.Name()
			go func() {
				userUUID := viper.GetString("UUID")
				if userUUID == "" {
					return
				}

				data := types.VersionTrackingData{
					UUID:      userUUID,
					Command:   commandName,
					Version:   debug.Version,
					Timestamp: time.Now().Format(time.RFC3339),
				}

				helpers.SendVersionTrackingData(data)
			}()
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		if debugFlag {
			debug.Debug = true
			helpers.Println("Debug mode on")
		}

		if versionFlag {
			helpers.Println("Version:", debug.Version)
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
	helpers.VtopLoginGlobal = vtop_login
	helpers.DecryptPasswordProxy = decryptPassword

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

	// Define flags for subcommands.
	marksCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	gradesCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	timeTableCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	holidayCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	holidayCmd.PersistentFlags().IntVarP(&classGrpFlag, "class-group", "g", 0, "Specify the class group")
	todayCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	todayCmd.PersistentFlags().IntVarP(&classGrpFlag, "class-group", "g", 0, "Specify the class group")
	tomorrowCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	tomorrowCmd.PersistentFlags().IntVarP(&classGrpFlag, "class-group", "g", 0, "Specify the class group")
	dayAfterCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	dayAfterCmd.PersistentFlags().IntVarP(&classGrpFlag, "class-group", "g", 0, "Specify the class group")
	examScheduleCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	calendarCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	calendarCmd.PersistentFlags().IntVarP(&classGrpFlag, "class-group", "g", 0, "Specify the class group")
	coursePageCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	coursePageCmd.PersistentFlags().IntVarP(&courseFlag, "course", "c", 0, "Specify the course")
	coursePageCmd.PersistentFlags().StringVarP(&facultyFlag, "faculty", "f", "", "Specify the faculty")
	coursePageCmd.PersistentFlags().IntVarP(&fuzzyIndexFlag, "fuzzy-index", "i", 0, "Specify the fuzzy index")
	coursePageArchiveCmd.PersistentFlags().IntVarP(&semesterFlag, "semester", "s", 0, "Specify the semester")
	coursePageArchiveCmd.PersistentFlags().IntVarP(&courseFlag, "course", "c", 0, "Specify the course")
	coursePageArchiveCmd.PersistentFlags().StringVarP(&facultyFlag, "faculty", "f", "", "Specify the faculty")
	coursePageArchiveCmd.PersistentFlags().IntVarP(&fuzzyIndexFlag, "fuzzy-index", "i", 0, "Specify the fuzzy index")
	syllabusCmd.PersistentFlags().StringVarP(&syllabusCourseFlag, "course", "c", "", "Specify course search query ")

	// Define global flags.
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Print Debug Messages")
	rootCmd.PersistentFlags().BoolVarP(&updateFlag, "update", "u", false, "Check for Updates")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print Version Number")

	// Add subcommands to root command.
	rootCmd.AddCommand(
		profileCmd,
		marksCmd,
		gradesCmd,
		attendanceCmd,
		timeTableCmd,
		holidayCmd,
		todayCmd,
		tomorrowCmd,
		dayAfterCmd,
		receiptCmd,
		hostelCmd,
		cgpaCmd,
		examScheduleCmd,
		libraryDuesCmd,
		logoutCmd,
		calendarCmd,
		coursePageCmd,
		coursePageArchiveCmd,
		nightslipCmd,
		leavestatusCmd,
		classMessagesCmd,
		daDetailsCmd,
		facilityCmd,
		eventsCmd,
		syllabusCmd,
		courseAllocationCmd,
		proxyCmd,
	)
}

func Execute() {
	startupArgs := inspectStartupArgs(os.Args[1:])

	if !shouldSkipStartupSideEffects(startupArgs) {
		killSwitch := helpers.CheckKillSwitch()
		if killSwitch == 2 {
			helpers.Println("This version of cli-top has been decommissioned.")
			return
		} else if killSwitch == 3 {
			err := helpers.OpenURLInBrowser("https://vtop.vit.ac.in")
			if err != nil {
				helpers.Println("An unexpected error has occurred", err)
			}
			return
		}

		err := godotenv.Load(configFilePath())
		if err != nil && debug.Debug {
			helpers.Println("Error loading .env file:", err)
		}

		userUUID := getOrCreateUUID()
		if debug.Debug {
			helpers.Println("User UUID:", userUUID)
		}

		if !startupArgs.rootUpdateRequested && helpers.IsInteractiveOutput(os.Stdout) {
			shouldNotify, latestVersion, highlight := helpers.ShouldShowUpdateNotification()
			if shouldNotify {
				helpers.ShowUpdateNotification(latestVersion, highlight)
			}
		}
	}

	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil && debug.Debug {
		helpers.Println(err)
		os.Exit(1)
	}
}

var courseAllocationCmd = &cobra.Command{
	Use:   "course-allocation",
	Short: "View course allocation",
	Run: helpers.CommandRunner("course-allocation", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.ExecuteInteractiveCourseAllocationView(regNo, cookies, "")
	}),
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show VTOP Student Profile",
	Run: helpers.CommandRunner("profile", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.Profile(cookies, regNo)
	}),
}

var facilityCmd = &cobra.Command{
	Use:   "facility",
	Short: "View facilities",
	Run: helpers.CommandRunner("facility", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.RegisterPhyFacility(regNo, cookies)
	}),
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "View upcoming club events and register for open ones",
	Run: helpers.CommandRunner("events", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetEvents(regNo, cookies)
	}),
}

var syllabusCmd = &cobra.Command{
	Use:   "syllabus",
	Short: "Download syllabus for a selected course",
	Run: helpers.CommandRunner("syllabus", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.ExecuteSyllabusDownload(regNo, cookies, syllabusCourseFlag)
	}),
}

var marksCmd = &cobra.Command{
	Use:   "marks",
	Short: "Show Marks Details of a particular semester",
	Run: helpers.CommandRunner("marks", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetMarks(regNo, cookies, "", semesterFlag)
	}),
}

var gradesCmd = &cobra.Command{
	Use:   "grades",
	Short: "Show Grade Details of a particular semester",
	Run: helpers.CommandRunner("grades", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetGrades(regNo, cookies, "", semesterFlag)
	}),
}

var attendanceCmd = &cobra.Command{
	Use:   "attendance",
	Short: "Show Attendance Details of a particular semester",
	Run: helpers.CommandRunner("attendance", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetAttendance(regNo, cookies, semesterFlag)
	}),
}

var receiptCmd = &cobra.Command{
	Use:   "receipts",
	Short: "Show Receipt Details of a user",
	Run: helpers.CommandRunner("receipts", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetReceipt(regNo, cookies)
	}),
}

var timeTableCmd = &cobra.Command{
	Use:   "timetable",
	Short: "Show Time Table of a particular semester",
	Run: helpers.CommandRunner("timetable", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetTimeTable(regNo, cookies, semesterFlag)
	}),
}

var holidayCmd = &cobra.Command{
	Use:   "holiday",
	Short: "Show upcoming class-impacting holidays for a semester",
	Run: helpers.CommandRunner("holiday", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetHolidayList(regNo, cookies, semesterFlag, classGrpFlag)
	}),
}

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's classes and whether attendance gives you room to skip them",
	Run: helpers.CommandRunner("today", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetToday(regNo, cookies, semesterFlag, classGrpFlag)
	}),
}

var tomorrowCmd = &cobra.Command{
	Use:   "tomorrow",
	Short: "Show tomorrow's classes and whether attendance gives you room to skip them",
	Run: helpers.CommandRunner("tomorrow", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetTomorrow(regNo, cookies, semesterFlag, classGrpFlag)
	}),
}

var dayAfterCmd = &cobra.Command{
	Use:     "dayafter",
	Aliases: []string{"day-after"},
	Short:   "Show the day after tomorrow's classes and whether attendance gives you room to skip them",
	Run: helpers.CommandRunner("dayafter", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetDayAfter(regNo, cookies, semesterFlag, classGrpFlag)
	}),
}

var hostelCmd = &cobra.Command{
	Use:   "hostel",
	Short: "Show Hostel Details of a user",
	Run: helpers.CommandRunner("hostel", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintHostelInfo(regNo, cookies, "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView")
	}),
}

var cgpaCmd = &cobra.Command{
	Use:   "cgpa",
	Short: "Show CGPA details",
	Run: helpers.CommandRunner("cgpa", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintCgpa(regNo, cookies, "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory")
	}),
}

var examScheduleCmd = &cobra.Command{
	Use:   "exams",
	Short: "Show Exam Schedule",
	Run: helpers.CommandRunner("exams", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetExamSchedule(regNo, cookies, semesterFlag)
	}),
}

var coursePageCmd = &cobra.Command{
	Use:   "course-page",
	Short: "Download course materials for a selected semester, course, and faculty",
	Run: helpers.CommandRunner("course-page", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.ExecuteCoursePageDownload(regNo, cookies, semesterFlag, courseFlag, facultyFlag, fuzzyIndexFlag)
	}),
}

var coursePageArchiveCmd = &cobra.Command{
	Use:   "course-page-archive",
	Short: "Download course materials for a selected semester, course, and faculty (Archive)",
	Run: helpers.CommandRunner("course-page-archive", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.ExecuteCoursePageOldDownload(regNo, cookies, semesterFlag, courseFlag, facultyFlag, fuzzyIndexFlag)
	}),
}

var libraryDuesCmd = &cobra.Command{
	Use:   "library-dues",
	Short: "Show Library Dues",
	Run: helpers.CommandRunner("library-dues", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetLibraryDues(regNo, cookies)
	}),
}

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Show Calendar",
	Run: helpers.CommandRunner("calendar", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintCal(regNo, cookies, semesterFlag, classGrpFlag)
	}),
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from VTOP",
	Run: helpers.CommandRunner("logout", func(cmd *cobra.Command, args []string) {
		err := godotenv.Load(configFilePath())
		helpers.LoadSemesterCacheFromEnv()
		if err != nil && debug.Debug {
			helpers.Println("Error loading .env file:", err)
			return
		}

		uuid := os.Getenv("UUID")

		if uuid == "" {
			helpers.Println("UUID not found; nothing to preserve.")
			return
		}

		env := map[string]string{
			"UUID": uuid,
		}

		// create the config file at the discovered path
		f, err := os.Create(configFilePath())
		if err != nil {
			if debug.Debug {
				helpers.Println("Error creating .env file:", err)
			}
			return
		}
		defer f.Close()

		for key, value := range env {
			_, err = f.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			if err != nil && debug.Debug {
				helpers.Println("Error writing to .env file:", err)
				return
			}
		}

		helpers.Println("Logged out successfully.")
	}),
}

var nightslipCmd = &cobra.Command{
	Use:   "nightslip",
	Short: "Show Nightslip Request Status of a user",
	Run: helpers.CommandRunner("nightslip", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetNightSlipStatus(regNo, cookies)
	}),
}

var leavestatusCmd = &cobra.Command{
	Use:   "leave",
	Short: "Show Leave Status",
	Run: helpers.CommandRunner("leave", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetLeaveStatus(regNo, cookies)
	}),
}

var classMessagesCmd = &cobra.Command{
	Use:   "msg",
	Short: "Show Class Messages",
	Run: helpers.CommandRunner("msg", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.GetClassMessage(regNo, cookies)
	}),
}

var daDetailsCmd = &cobra.Command{
	Use:   "da",
	Short: "Show Digital Assignment Details",
	Run: helpers.CommandRunner("da", func(cmd *cobra.Command, args []string) {
		cookies, regNo := readCookiesFromFile()
		features.PrintAllDAs(regNo, cookies, courseNameFlag)
	}),
}
