package main

import (
	"net/http"
	"time"
)

// httpRequestTimeout bounds how long glow will wait on a single remote fetch.
// Without it, a slow or unresponsive host (for example when resolving a
// github://, gitlab:// or https:// source) could make glow hang indefinitely.
const httpRequestTimeout = 30 * time.Second

// httpClient is the shared HTTP client used for all outbound requests. It sets
// a timeout so that an unresponsive remote host cannot stall glow forever.
var httpClient = &http.Client{
	Timeout: httpRequestTimeout,
}
