package helpers

import (
	"crypto/tls"
	"net/http"
	"time"
)

var sharedHTTPClient *http.Client

func init() {
	sharedHTTPClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// GetHTTPClient returns the shared HTTP client used throughout the application.
func GetHTTPClient() *http.Client {
	return sharedHTTPClient
}
