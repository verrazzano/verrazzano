// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cors

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

var allowedOriginsWhitelistFunc = allowedOriginsWhitelist

func AddCORSHeaders(req *http.Request, rw http.ResponseWriter, ingressHost string) (int, error) {
	var origin string
	var err error
	if origin, err = getOriginHeaderValue(req); err != nil {
		return http.StatusBadRequest, err
	}
	// nothing to do for CORS
	if origin == "" {
		return http.StatusOK, nil
	}

	allowed := originAllowed(origin, ingressHost)

	// From https://tools.ietf.org/id/draft-abarth-origin-03.html#server-behavior, if the request Origin is not in
	// whitelisted origins and the request method is not a safe non state changing (i.e. GET or HEAD), we should
	// abort the request.
	if !allowed && req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
		// TODO forbidden doesn't seem right here, but that's what current authproxy does
		return http.StatusForbidden, fmt.Errorf("Origin %s is not allowed", origin)
	}

	// Add response headers if it's an allowed origin
	if allowed {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
		rw.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if req.Method == http.MethodOptions {
		rw.Header().Set("Access-Control-Allow-Headers", "authorization, content-type")
		rw.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH")
	}
	return http.StatusOK, nil
}

func getOriginHeaderValue(req *http.Request) (string, error) {
	originValues := req.Header["Origin"]
	if len(originValues) > 1 {
		return "", fmt.Errorf("Origin header must have a single value")
	}
	if len(originValues) == 0 {
		return "", nil
	}
	origin := originValues[0]
	if origin == "*" {
		// not a legit origin, could be intended to trick us into oversharing
		return origin, fmt.Errorf("Invalid Origin header: '*'")
	}
	return origin, nil
}

// originAllowed validates the origin string and returns true if it is an allowed value
func originAllowed(origin string, ingressHost string) bool {
	// Origin may be set to "null" in private-sensitive contexts as defined by the application,
	// according to https://datatracker.ietf.org/doc/rfc6454
	if origin == "null" {
		return false
	}

	ingressURL := "https://" + ingressHost
	if origin == ingressURL {
		return true
	}

	// Check whitelist of allowed origins if provided
	var allowedOriginsStr string
	if allowedOriginsStr = allowedOriginsWhitelistFunc(); allowedOriginsStr == "" {
		return false
	}
	allowedOrigins := strings.Split(allowedOriginsStr, ",")
	for _, allowed := range allowedOrigins {
		if origin == strings.TrimSpace(allowed) {
			return true
		}
	}

	return false
}

func allowedOriginsWhitelist() string {
	return os.Getenv("VZ_API_ALLOWED_ORIGINS")
}
