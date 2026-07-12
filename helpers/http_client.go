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

	sessionCache := tls.NewLRUClientSessionCache(128)

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxConnsPerHost:       128,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ReadBufferSize:        32 * 1024,
		WriteBufferSize:       32 * 1024,
		TLSClientConfig: &tls.Config{
			ClientSessionCache: sessionCache,
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
