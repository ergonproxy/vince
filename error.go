package main

import (
	"net/http"
	"sync"

	"github.com/ergongate/vince/templates"
)

// source https://en.wikipedia.org/wiki/List_of_HTTP_status_codes#nginx
const (
	// Used internally[89] to instruct the server to return no information to the
	// client and close the connection immediately.
	statusNoResponse = 444
	// Client sent too large request or too long header line.
	statusRequestHeaderTooLarge = 494
	// An expansion of the 400 Bad Request response code, used when the client has
	// provided an invalid client certificate.
	statusSSLCertificateError = 495
	// An expansion of the 400 Bad Request response code, used when a client
	// certificate is required but not provided.
	statusSSLCertificateRequired = 496
	// An expansion of the 400 Bad Request response code, used when the client has
	// made a HTTP request to a port listening for HTTPS requests.
	statusHTTPToHTTPSPort = 497
	// Used when the client has closed the request before the server could send a
	// response.
	statusClientClosedRequest = 499
)

var statusTextMap = map[int]string{
	statusNoResponse:             "No Response",
	statusRequestHeaderTooLarge:  "Request header too large",
	statusSSLCertificateError:    "SSL Certificate Error",
	statusSSLCertificateRequired: "SSL Certificate Required",
	statusHTTPToHTTPSPort:        "HTTP Request Sent to HTTPS Port",
	statusClientClosedRequest:    "Client Closed Request",
}

var statusCodesLock sync.Mutex

func statusText(code int) string {
	if code >= 444 && code <= 499 {
		statusCodesLock.Lock()
		txt, ok := statusTextMap[code]
		statusCodesLock.Unlock()
		if ok {
			return txt
		}
	}
	return http.StatusText(code)
}

func e500(w http.ResponseWriter) {
	eRender(w, http.StatusInternalServerError)
}

func e404(w http.ResponseWriter) error {
	return eRender(w, http.StatusNotFound)
}

func eRender(w http.ResponseWriter, code int) error {
	w.WriteHeader(code)
	return templates.ExecHTML(w, "errors/error.html", map[string]interface{}{
		"code": code,
		"text": statusText(code),
	})
}
