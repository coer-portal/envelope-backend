package router

import (
	"math/rand"
	"net/http"
	"strings"
)

// parseForm parses the form in a request and handles the error appropriately
func parseForm() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {
		err := r.ParseForm()

		if err != nil {
			return &HTTPError{
				IError:    err,
				Level:     1,
				Status:    http.StatusBadRequest,
				ErrorCode: ErrParsing,
			}
		}
		return nil
	}
}

// parseDeviceID parses "deviceid" from query parameters in a GET request and from Form value in a POST request
func parseDeviceID() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		var deviceid string

		if r.Method == "POST" {
			deviceid = r.Form.Get("deviceid")
			if deviceid == "" {
				return handleMissingDataError("deviceid")
			}
		} else {
			deviceid = r.URL.Query().Get("deviceid")
			if deviceid == "" {
				return handleMissingDataError("deviceid")
			}
		}
		rc.deviceid = deviceid
		return nil
	}
}

func handleJSONError(err error) *HTTPError {
	return &HTTPError{
		ErrorCode: ErrInternal,
		IError:    err,
		Level:     3,
		Status:    http.StatusInternalServerError,
	}
}

// handleMissingDataError takes name of data that is missing or invalid and return *HTTPError
func handleMissingDataError(v string) *HTTPError {
	return &HTTPError{
		Level:     1,
		Status:    http.StatusBadRequest,
		ErrorCode: ErrNotFound,
	}
}

func fetchRemoteIpAddr(ip string) string {
	if strings.Contains(ip, "[::1]") {
		return "127.0.0.1"
	}
	return ip
}

// input: postid; output: post, timestamp, likes, comments
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
