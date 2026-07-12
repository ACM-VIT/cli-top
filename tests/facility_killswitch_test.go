package tests

import (
	"net/http/httptest"
	"testing"

	"cli-top/features"
	"cli-top/helpers"
)

func TestFacilityRegistrationFailsClosedWhenKillSwitchIsUnavailable(t *testing.T) {
	originalLatestJSONURL := helpers.GetLatestJSONURL()
	t.Cleanup(func() { helpers.SetLatestJSONURL(originalLatestJSONURL) })

	server := httptest.NewServer(nil)
	serverURL := server.URL
	server.Close()

	t.Setenv(latestJSONURLEnv, serverURL)
	helpers.SetLatestJSONURL(originalLatestJSONURL)
	transport := installSharedWorkflowTransport(t, facilityWorkflowResponse)

	if err := features.RegisterPhyFacility(interactiveFeatureRegNo, interactiveFeatureCookies, "Basketball", true); err != nil {
		t.Fatalf("RegisterPhyFacility returned error: %v", err)
	}

	assertWorkflowTransportClean(t, transport)
	if transport.submitted {
		t.Fatal("facility registration was submitted while kill-switch metadata was unavailable")
	}
	requireWorkflowRequest(t, transport, "/vtop/phyedu/facilityAvailable", 2)
	for _, request := range transport.requests {
		if request.path == "/vtop/phyedu/PhyFacilityProcessRegistration" {
			t.Fatalf("unexpected facility registration request: %#v", request)
		}
	}
}
