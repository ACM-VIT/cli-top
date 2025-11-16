package helpers

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const defaultRequestTimeout = 45 * time.Second

var sharedHTTPClient *http.Client

func init() {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 60 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxConnsPerHost:       64,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	sharedHTTPClient = &http.Client{
		Timeout:   10 * time.Minute, // precise limits enforced via per-request contexts
		Transport: transport,
	}
}

// GetHTTPClient returns the shared HTTP client used throughout the application.
func GetHTTPClient() *http.Client {
	return sharedHTTPClient
}
