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

func RegisterPhyFacility(regNo string, cookies types.Cookies) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login using the cli-top login command.")
		return
	}

	facilities, err := fetchAvailableFacilities(regNo, cookies)
	if err != nil {
		fmt.Println("Error fetching facilities:", err)
		return
	}

	if len(facilities) == 0 {
		fmt.Println("No facilities found.")
		return
	}

	displayFacilities(facilities)

	selectedFacility, err := promptFacilitySelection(facilities)
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

	doc.Find("table.table-bordered.table-hover.table-stripped.dataTable tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		cells := s.Find("td")
		if cells.Length() < 4 {
			return
		}

		name := strings.TrimSpace(cells.Eq(0).Text())
		feesStr := strings.TrimSpace(cells.Eq(1).Text())
		seatsStr := strings.TrimSpace(cells.Eq(2).Text())
		actionCell := cells.Eq(3)

		fees := feesStr

		seatsAvailable, err := strconv.Atoi(seatsStr)
		if err != nil {
			seatsAvailable = 0
		}

		actionHTML, _ := actionCell.Html()
		matches := buttonRegex.FindStringSubmatch(actionHTML)
		var miscID string
		if len(matches) == 2 {
			miscID = matches[1]
		} else {
			miscID = ""
			if debug.Debug {
				fmt.Printf("Unable to extract miscID for facility: %s\n", name)
			}
		}

		facility := types.Facility{
			ID:             "1",    
			Name:           name,
			Fees:           fees,
			SeatsAvailable: seatsAvailable,
			MiscID:         miscID,
		}

		facilities = append(facilities, facility)
	})

	if debug.Debug {
		fmt.Printf("Parsed %d facilities.\n", len(facilities))
		for _, f := range facilities {
			fmt.Printf("Facility: %s, Fees: %s, Seats Available: %d, MiscID: %s\n",
				f.Name, f.Fees, f.SeatsAvailable, f.MiscID)
		}
	}

	return facilities, nil
}

func displayFacilities(facilities []types.Facility) {
	nestedList := [][]string{
		{"No.", "Facility Name", "Fees (Including GST)", "Seats Available"},
	}

	for i, facility := range facilities {
		var seatsStr string
		if facility.MiscID != "" && facility.SeatsAvailable > 0 {
			seatsStr = strconv.Itoa(facility.SeatsAvailable)
		} else {
			seatsStr = Colorize("Full", "red")
		}
		nestedList = append(nestedList, []string{
			strconv.Itoa(i + 1),
			facility.Name,
			facility.Fees,
			seatsStr,
		})
	}

	fmt.Println()
	helpers.PrintTable(nestedList, 2)
	fmt.Println()
}

func promptFacilitySelection(facilities []types.Facility) (types.Facility, error) {
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
		if selectedFacility.MiscID == "" || selectedFacility.SeatsAvailable <= 0 {
			fmt.Println("Selected facility is full. Please choose another facility.")
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

	confirmationMsg := doc.Find("div.alert").Text()
	confirmationMsg = helpers.SanitizeString(confirmationMsg) 

	if confirmationMsg == "" {
		fmt.Println("No confirmation message found. Check if the request was successful through another method.")
		return fmt.Errorf("no confirmation message found")
	}

	fmt.Println("Facility Registration Response:")
	fmt.Println(confirmationMsg)

	return nil
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
