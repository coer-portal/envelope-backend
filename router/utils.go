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
				IError:          err,
				Level:           1,
				GenericResponse: HTTPResponse(http.StatusBadRequest),
				ErrorCode:       ErrParsing,
			}
		}
		return nil
	}
}

// verifyDeviceID is a middleware that can be plugged in to make sure the specified endpoint is only accessible to registered users
func verifyDeviceID() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		_, err := rc.db.VerifyDeviceID(rc.ctx, rc.deviceid)

		if err != nil {
			if err.Error() == ErrNotRegistered {
				return &HTTPError{
					ErrorCode:       ErrNotRegistered,
					GenericResponse: HTTPResponse(http.StatusUnauthorized),
					Level:           1,
				}
			}

			return &HTTPError{
				deviceid:        rc.deviceid,
				ErrorCode:       ErrInternal,
				IError:          err,
				Level:           3,
				GenericResponse: HTTPResponse(http.StatusInternalServerError),
			}
		}

		return nil
	}
}

func handleJSONError(err error) *HTTPError {
	return &HTTPError{
		ErrorCode:       ErrInternal,
		IError:          err,
		Level:           3,
		GenericResponse: HTTPResponse(http.StatusInternalServerError),
	}
}

// handleMissingDataError takes name of data that is missing or invalid and return *HTTPError
func handleMissingDataError(v string) *HTTPError {
	return &HTTPError{
		Level:           1,
		ErrorCode:       ErrNotFound,
		GenericResponse: HTTPResponse(http.StatusBadRequest),
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
