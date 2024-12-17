package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// RegisterPhyFacility initiates the registration process for the Physical Education Facility.
// It first sends a POST request to the facilityAvailable endpoint to simulate availability check,
// then proceeds to register the facility.
func RegisterPhyFacility(regNo string, cookies types.Cookies) {
	// Ensure that we have valid session data
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login using the cli-top login command.")
		return
	}

	// Step 1: Simulate Facility Availability Check
	err := simulateFacilityAvailability(regNo, cookies)
	if err != nil {
		fmt.Println("Failed to simulate facility availability check. Registration aborted.")
		return
	}

	// Step 2: Proceed with Facility Registration
	registerFacility(regNo, cookies)
}

// simulateFacilityAvailability sends a POST request to the facilityAvailable endpoint.
// It does not process the response as per the requirements.
func simulateFacilityAvailability(regNo string, cookies types.Cookies) error {
	// URL for facility availability simulation
	url := "https://vtop.vit.ac.in/vtop/phyedu/facilityAvailable"

	// Generate the 'nocache' timestamp in milliseconds since epoch
	nocache := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Construct the payload with dynamic 'nocache' value
	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%s",
		regNo,
		cookies.CSRF,
		nocache,
	)

	// Make the POST request
	_, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error simulating facility availability:", err)
		}
		return err
	}

	if debug.Debug {
		fmt.Println("Facility availability check simulated successfully.")
	}

	return nil
}

// registerFacility sends the POST request to register for the Physical Education Facility.
// It retains the original functionality of parsing and displaying the confirmation message.
func registerFacility(regNo string, cookies types.Cookies) {
	// URL for the physical education facility registration
	url := "https://vtop.vit.ac.in/vtop/phyedu/PhyFacilityProcessRegistration"

	// Generate the 'x' timestamp dynamically in UTC RFC1123 format
	xTime := time.Now().UTC().Format(time.RFC1123)

	// Construct the payload with the provided values
	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&x=%s&facilityId=%s&miscId=%s",
		cookies.CSRF,
		regNo,
		xTime,
		"1",   // facilityId
		"489", // miscId
	)

	// Make the POST request
	body, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error initiating physical education facility registration:", err)
		}
		return
	}

	// Print the full response body for debugging
	if debug.Debug {
		fmt.Println("Full Response Body:")
		fmt.Println(string(body))
		fmt.Println("--------------------------------------------------")
	}

	// Parse the response if it's HTML; adjust accordingly if the response isn't HTML.
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML response:", err)
		}
		return
	}

	// Attempt to extract a confirmation message from the response
	confirmationMsg := doc.Find("div.alert").Text()
	confirmationMsg = helpers.SanitizeString(confirmationMsg) // Using the SanitizeString function

	if confirmationMsg == "" {
		fmt.Println("No confirmation message found. Check if the request was successful through another method.")
		return
	}

	fmt.Println("Facility Registration Response:")
	fmt.Println(confirmationMsg)
}
