package features

import (
	"bufio"
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	FacilityTableSelector        = "table.table-bordered.table-hover.table-stripped.dataTable"
	FacilityRowsSelector         = "tr"
	FacilityCellSelector         = "td"
	FacilityPanelHeadingSelector = "div.panel-heading.panel-head-custom"
)

// RegisterPhyFacility fetches and displays physical education facilities available for registration.
// When FacilityRegistrationEnabled is set to 1, it allows interactive registration.
// When FacilityRegistrationEnabled is set to 0, it only displays the facilities without prompting for registration.
func RegisterPhyFacility(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	killSwitch := helpers.CheckKillSwitch()
	if killSwitch == 4 {
		// fmt.Println("This feature is currently disabled by the administrator (killswitch=4). View-only mode enabled.")
		registrations, err := ListRegistrations(regNo, cookies)
		if err != nil {
			fmt.Println("Error fetching registrations:", err)
			registrations = []types.Registration{}
		}
		facilities, err := fetchAvailableFacilities(regNo, cookies)
		if err != nil {
			fmt.Println("Error fetching facilities:", err)
		}
		displayFacilities(facilities, registrations)
		return
	}

	registrations, err := ListRegistrations(regNo, cookies)
	if err != nil {
		fmt.Println("Error fetching registrations:", err)
		registrations = []types.Registration{}
	}

	facilities, err := fetchAvailableFacilities(regNo, cookies)
	if err != nil {
		fmt.Println("Error fetching facilities:", err)
	}

	if len(facilities) == 0 && len(registrations) == 0 {
		fmt.Println("No facilities or registrations found.")
		return
	}
	if err != nil {
		fmt.Println("Error fetching registrations:", err)
		registrations = []types.Registration{}
	}
	for _, reg := range registrations {
		found := false
		for idx, fac := range facilities {
			if strings.EqualFold(strings.TrimSpace(fac.Name), strings.TrimSpace(reg.FacilityName)) {
				facilities[idx].Registered = true
				found = true
				break
			}
		}
		if !found {
			facilities = append(facilities, types.Facility{
				ID:             "",
				Name:           reg.FacilityName,
				Fees:           "",
				SeatsAvailable: 0,
				MiscID:         "",
				Registered:     true,
			})
		}
	}

	displayFacilities(facilities, registrations)

	if killSwitch == 4 {
		// fmt.Println("Registration feature is currently in view-only mode.")
		return
	}

	selectedFacility, err := promptFacilitySelection(facilities, nil)
	if err != nil {
		fmt.Println("Registration aborted:", err)
		return
	}

	err = performRegistration(regNo, cookies, selectedFacility)
	if err != nil {
		fmt.Println("Error during registration:", err)
		return
	}

	fmt.Println("Registration completed successfully.")

	updatedRegistrations, err := ListRegistrations(regNo, cookies)
	if err != nil {
		fmt.Println("Error fetching updated registrations:", err)
		return
	}

	for _, reg := range updatedRegistrations {
		for idx, fac := range facilities {
			if strings.EqualFold(strings.TrimSpace(fac.Name), strings.TrimSpace(reg.FacilityName)) {
				facilities[idx].Registered = true
				break
			}
		}
	}

	fmt.Println("\nYour Current Registrations:")
	displayFacilities(facilities, updatedRegistrations)
}

func fetchAvailableFacilities(regNo string, cookies types.Cookies) ([]types.Facility, error) {
	url := "https://vtop.vit.ac.in/vtop/phyedu/facilityAvailable"

	nocache := fmt.Sprintf("%d", time.Now().UnixMilli())

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%s",
		regNo,
		cookies.CSRF,
		nocache,
	)

	body, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching facilities:", err)
		}
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML response:", err)
		}
		return nil, err
	}

	var facilities []types.Facility

	buttonRegex := regexp.MustCompile(`registerNow\(["']1["'],\s*["'](\d+)["']\)`)

	doc.Find(FacilityTableSelector).Find(FacilityRowsSelector).Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			cells := s.Find(FacilityCellSelector)
			if cells.Length() >= 1 {
				firstCell := strings.ToLower(strings.TrimSpace(cells.Eq(0).Text()))
				if firstCell == "facility name" || firstCell == "facility" {
					return
				}
			}
		}

		cells := s.Find(FacilityCellSelector)
		if cells.Length() < 4 {
			if debug.Debug {
				fmt.Printf("Skipping row %d: insufficient cells\n", i)
			}
			return
		}

		name := strings.TrimSpace(cells.Eq(0).Text())
		feesStr := strings.TrimSpace(cells.Eq(1).Text())
		seatsStr := strings.TrimSpace(cells.Eq(2).Text())
		actionCell := cells.Eq(3)

		seatsAvailable, err := strconv.Atoi(seatsStr)
		if err != nil {
			seatsAvailable = 0
		}

		onclick, exists := actionCell.Find("button").Attr("onclick")
		if !exists {
			onclick = ""
			if debug.Debug {
				fmt.Printf("No onclick attribute found for facility: %s\n", name)
			}
		}

		if debug.Debug {
			fmt.Printf("Facility: %s, onclick attribute: %s\n", name, onclick)
		}

		matches := buttonRegex.FindStringSubmatch(onclick)
		var miscID string
		if len(matches) == 2 {
			miscID = matches[1]
			if debug.Debug {
				fmt.Printf("Extracted miscID: %s for facility: %s\n", miscID, name)
			}
		} else {
			miscID = ""
			if debug.Debug {
				fmt.Printf("Unable to extract miscID for facility: %s\n", name)
			}
		}

		facility := types.Facility{
			ID:             "1",
			Name:           name,
			Fees:           feesStr,
			SeatsAvailable: seatsAvailable,
			MiscID:         miscID,
			Registered:     false,
		}

		facilities = append(facilities, facility)
	})

	if debug.Debug {
		fmt.Printf("Parsed %d facilities.\n", len(facilities))
		for _, f := range facilities {
			fmt.Printf("Facility: %s, Fees: %s, Seats Available: %d, MiscID: %s, Registered: %v\n",
				f.Name, f.Fees, f.SeatsAvailable, f.MiscID, f.Registered)
		}
	}

	return facilities, nil
}

func ListRegistrations(regNo string, cookies types.Cookies) ([]types.Registration, error) {
	url := "https://vtop.vit.ac.in/vtop/phyedu/facilityAvailable"

	nocache := fmt.Sprintf("%d", time.Now().UnixMilli())

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%s",
		regNo,
		cookies.CSRF,
		nocache,
	)

	body, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching registrations:", err)
		}
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML response:", err)
		}
		return nil, err
	}

	var registrations []types.Registration

	registrationsHeader := doc.Find(FacilityPanelHeadingSelector).FilterFunction(func(i int, s *goquery.Selection) bool {
		return strings.TrimSpace(s.Text()) == "My Registration(s)"
	})

	if registrationsHeader.Length() == 0 {
		if debug.Debug {
			fmt.Println("Registration section not found in the response.")
		}
		return registrations, nil
	}

	registrationsTable := registrationsHeader.NextAllFiltered("div.box-body").Find("table.dataTable").First()
	if registrationsTable.Length() == 0 {
		if debug.Debug {
			fmt.Println("Registration table not found in the response.")
		}
		return registrations, nil
	}

	registrationsTable.Find(FacilityRowsSelector).Each(func(i int, s *goquery.Selection) {
		cells := s.Find(FacilityCellSelector)
		if cells.Length() < 2 {
			return
		}
		facilityName := strings.TrimSpace(cells.Eq(0).Text())
		statusMessage := strings.TrimSpace(cells.Eq(1).Text())

		isPaid := strings.Contains(statusMessage, "Paid") && !strings.Contains(statusMessage, "not paid")

		registration := types.Registration{
			FacilityName:  facilityName,
			StatusMessage: statusMessage,
			IsPaid:        isPaid,
		}

		registrations = append(registrations, registration)
	})
	if debug.Debug {
		fmt.Printf("Parsed %d registrations.\n", len(registrations))
		for _, reg := range registrations {
			fmt.Printf("Registered Facility: %s, Status: %s, Paid: %v\n", reg.FacilityName, reg.StatusMessage, reg.IsPaid)
		}
	}

	return registrations, nil
}

func displayFacilities(facilities []types.Facility, registrations []types.Registration) {
	if len(facilities) == 0 && len(registrations) > 0 {
		fmt.Println("\nYour Current Registrations:")
		nestedList := [][]string{
			{"No.", "Facility Name", "Status"},
		}

		for i, reg := range registrations {
			statusStr := ""
			if reg.IsPaid {
				statusStr = Colorize("Registered (Paid)", "green")
			} else {
				statusStr = Colorize("Registered (Not Paid)", "yellow")
			}

			nestedList = append(nestedList, []string{
				strconv.Itoa(i + 1),
				reg.FacilityName,
				statusStr,
			})
		}

		fmt.Println()
		helpers.PrintTable(nestedList, 2)
		fmt.Println()
		return
	}

	nestedList := [][]string{
		{"No.", "Facility Name", "Fees (Including GST)", "Status"},
	}
	for i, facility := range facilities {
		var statusStr string
		if facility.Registered {
			var isPaid bool
			for _, reg := range registrations {
				if strings.EqualFold(strings.TrimSpace(reg.FacilityName), strings.TrimSpace(facility.Name)) {
					isPaid = reg.IsPaid
					break
				}
			}

			if isPaid {
				statusStr = Colorize("Registered (Paid)", "green")
			} else {
				statusStr = Colorize("Registered (Not Paid)", "yellow")
			}
		} else {
			if facility.SeatsAvailable > 0 {
				statusStr = fmt.Sprintf("%d seats left", facility.SeatsAvailable)
			} else {
				statusStr = Colorize("Full", "red")
			}
		}

		nestedList = append(nestedList, []string{
			strconv.Itoa(i + 1),
			facility.Name,
			facility.Fees,
			statusStr,
		})
	}

	fmt.Println()
	helpers.PrintTable(nestedList, 2)
	fmt.Println()
}

func promptFacilitySelection(facilities []types.Facility, registrationsMap map[string]bool) (types.Facility, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter the number of the facility you want to register for (or type 'exit' to cancel): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if debug.Debug {
				fmt.Println("Error reading input:", err)
			}
			return types.Facility{}, fmt.Errorf("failed to read input")
		}

		input = strings.TrimSpace(input)
		if strings.ToLower(input) == "exit" {
			return types.Facility{}, fmt.Errorf("user canceled the selection")
		}

		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(facilities) {
			fmt.Println("Invalid selection. Please enter a valid facility number.")
			continue
		}

		selectedFacility := facilities[selection-1]
		if selectedFacility.Registered {
			fmt.Println("You are already registered for this facility.")
			continue
		}
		if selectedFacility.MiscID == "" || selectedFacility.SeatsAvailable <= 0 {
			fmt.Println("Selected facility is full or cannot be registered. Please choose another facility.")
			continue
		}

		fmt.Printf("You have selected '%s' with %d seats available.\n", selectedFacility.Name, selectedFacility.SeatsAvailable)
		fmt.Print("Do you want to proceed with registration? (yes/no): ")
		confirmInput, err := reader.ReadString('\n')
		if err != nil {
			if debug.Debug {
				fmt.Println("Error reading confirmation:", err)
			}
			return types.Facility{}, fmt.Errorf("failed to read confirmation")
		}

		confirmInput = strings.ToLower(strings.TrimSpace(confirmInput))
		if confirmInput == "yes" || confirmInput == "y" {
			return selectedFacility, nil
		} else if confirmInput == "no" || confirmInput == "n" {
			return types.Facility{}, fmt.Errorf("user declined the registration")
		} else {
			fmt.Println("Invalid input. Please respond with 'yes' or 'no'.")
			continue
		}
	}
}

func performRegistration(regNo string, cookies types.Cookies, facility types.Facility) error {
	if facility.ID == "" || facility.MiscID == "" {
		fmt.Println("Cannot proceed with registration due to missing facility identifiers.")
		return fmt.Errorf("missing facility identifiers")
	}

	url := "https://vtop.vit.ac.in/vtop/phyedu/PhyFacilityProcessRegistration"

	xTime := time.Now().UTC().Format(time.RFC1123)

	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&x=%s&facilityId=%s&miscId=%s",
		cookies.CSRF,
		regNo,
		xTime,
		facility.ID,
		facility.MiscID,
	)

	body, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error initiating physical education facility registration:", err)
		}
		return err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML response:", err)
		}
		return err
	}

	registrationsHeader := doc.Find("div.panel-heading.panel-head-custom").FilterFunction(func(i int, s *goquery.Selection) bool {
		return strings.TrimSpace(s.Text()) == "My Registration(s)"
	})

	if registrationsHeader.Length() == 0 {
		fmt.Println("Registration confirmation section not found in the response.")
		return fmt.Errorf("registration confirmation section not found")
	}

	registrationsTable := registrationsHeader.NextAllFiltered("div.box-body").Find("table.dataTable").First()
	if registrationsTable.Length() == 0 {
		fmt.Println("Registration table not found in the response. Unable to verify registration.")
		return fmt.Errorf("registration table not found")
	}

	registrationSuccess := false
	confirmationMessage := ""

	registrationsTable.Find("tr").Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 2 {
			return
		}

		facilityName := strings.TrimSpace(cells.Eq(0).Text())
		statusMessage := strings.TrimSpace(cells.Eq(1).Text())

		if facilityName == facility.Name {
			registrationSuccess = true
			confirmationMessage = helpers.SanitizeString(statusMessage)
		}
	})

	if registrationSuccess {
		fmt.Println("Facility Registration Response:")
		fmt.Println(confirmationMessage)
		return nil
	} else {
		fmt.Println("Registration might have failed. Confirmation message not found.")
		return fmt.Errorf("registration confirmation not found for facility: %s", facility.Name)
	}
}

func Colorize(text string, color string) string {
	colorCodes := map[string]string{
		"black":   "\033[30m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"reset":   "\033[0m",
	}

	if code, exists := colorCodes[strings.ToLower(color)]; exists {
		return code + text + colorCodes["reset"]
	}

	return text
}
