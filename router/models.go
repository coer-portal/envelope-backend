package router

import (
	"net/http"
)

type SubmitPostResponse struct {
	PostID    int   `json:"postid"`
	Timestamp int64 `json:"timestamp"`
	likes     int   `json:"likes"`
	GenericResponse
}

type GenericResponse struct {
	Status string `json:"status"`
	Code   int    `json:"status_code"`
}

type RegisterDeviceResponse struct {
	Hash string `json:"hash"`
	GenericResponse
}

// HTTPError is returned by middlewares
// This is used internally by the application
// and some fields are serialized and sent as error response to the request.
type HTTPError struct {
	// Error Level, Used Internally
	Level int `json:"-"`
	//Error message that's logged to console
	IError error `json:"-"`
	// Device ID of request Origin
	deviceid string `json:"-"`
	// Short Error Code that can be used by client to pinpoint exact error
	ErrorCode string `json:"error_code"`

	GenericResponse
}

func (e HTTPError) Error() string {
	return e.IError.Error()
}

func HTTPResponse(code int) GenericResponse {
	return GenericResponse{Status: http.StatusText(code), Code: code}
}
