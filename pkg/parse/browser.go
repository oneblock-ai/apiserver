package parse

import (
	"strings"

	"github.com/cloudwego/hertz/pkg/protocol"
)

func IsBrowser(req protocol.Request, checkAccepts bool) bool {
	accepts := strings.ToLower(req.Header.Get("Accept"))
	userAgent := strings.ToLower(req.Header.Get("User-Agent"))

	if accepts == "" || !checkAccepts {
		accepts = "*/*"
	}

	// User agent has Mozilla and browser accepts */*
	return strings.Contains(userAgent, "mozilla") && strings.Contains(accepts, "*/*")
}

//func MatchNotBrowser(req *http.Request, match *mux.RouteMatch) bool {
//	return !MatchBrowser(req, match)
//}
//
//func MatchBrowser(req *http.Request, _ *mux.RouteMatch) bool {
//	return IsBrowser(req, true)
//}
