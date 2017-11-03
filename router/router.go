package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ishanjain28/envelope-backend/common"
	"github.com/ishanjain28/envelope-backend/db"
	"github.com/ishanjain28/envelope-backend/log"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (

	// ErrNotFound is sent when an expected resource in request is unavailable
	ErrNotFound         = "NOTFOUND"
	ErrNotRegistered    = "NOTREGISTERED"
	ErrInternal         = "INTERNALERROR"
	ErrOutOfValidRegion = "OUTOFREGION"
	ErrMismatch         = "MISMATCH"
	ErrParsing          = "PARSINGERROR"
	workingRegion       = "Uttarakhand"
)

type RouterContext struct {
	rclient  *redis.Client
	pqdb     *db.DB
	deviceid string
}

type HTTPError struct {
	Level     int    `json:"-"`
	IError    error  `json:"-"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

type Handler func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError

func Handle(client *redis.Client, pqdb *db.DB, handlers ...Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rc := &RouterContext{
			rclient: client,
			pqdb:    pqdb,
		}
		w.Header().Add("Content-Type", "application/json")

		for _, handler := range handlers {
			e := handler(rc, w, r)
			if e != nil {

				// 3 Levels of errors
				// Level 1: Don't log anything on server, Only return a response to the user
				// Level 2: Log the error as warning on the server, But don't send a response or close the request
				// Level 3: Log the request, Cancel the request from going any further and return an appropriate response
				switch e.Level {
				case 1:
					w.WriteHeader(e.Status)
					err := json.NewEncoder(w).Encode(e)
					if err != nil {
						w.Header().Set("Content-Type", "text/plain")
						w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
					}
					return

				case 2:
					log.Warn.Printf("%v: %s\n", e.IError, e.IError)

				case 3:
					w.WriteHeader(e.Status)
					err := json.NewEncoder(w).Encode(e)
					if err != nil {
						log.Error.Printf("%v: %s\n", err, err)
						w.Header().Set("Content-Type", "text/plain")
						w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
					}
					return
				}
			}
		}
	})
}

func Init(client *redis.Client, pqdb *db.DB) *mux.Router {
	r := mux.NewRouter()

	r.Handle("/register-device", Handle(client, nil,
		parseForm(),
		parseDeviceID(),
		RegisterDevice(),
	)).Methods("POST")

	r.Handle("/verify-device", Handle(client, nil,
		parseDeviceID(),
		VerifyDevice(),
	)).Methods("GET")

	r.Handle("/report", Handle(client, pqdb,
		parseForm(),
		parseDeviceID(),
		report(),
	)).Methods("POST")

	return r
}

// parseForm parses the form in a request and handles the error appropriately
func parseForm() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {
		err := r.ParseForm()

		if err != nil {
			return &HTTPError{
				IError:    err,
				Level:     1,
				Status:    http.StatusBadRequest,
				Error:     "error in parsing form",
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
				return &HTTPError{
					Level:     1,
					Error:     "No Device Id",
					ErrorCode: ErrNotFound,
					Status:    http.StatusBadRequest,
				}
			}
		} else {
			deviceid = r.URL.Query().Get("deviceid")
			if deviceid == "" {
				return &HTTPError{
					Level:     1,
					Error:     "No Device Id",
					ErrorCode: ErrNotFound,
					Status:    http.StatusBadRequest}
			}
		}

		rc.deviceid = deviceid
		return nil
	}
}

// RegisterDevice receives a deviceid via POST and puts it in redis for 2 months, And sends a Hash back in response
func RegisterDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		region, err := common.GetRegionofIP(r.RemoteAddr)
		if err != nil {
			return &HTTPError{
				ErrorCode: ErrInternal,
				Error:     "error in registering device",
				Level:     3,
				Status:    http.StatusInternalServerError,
			}
		}

		if region != workingRegion {
			return &HTTPError{
				Error:     "Out of working region",
				ErrorCode: ErrOutOfValidRegion,
				IError:    errors.New(fmt.Sprintf("%s: %s is from %s", ErrOutOfValidRegion, r.RemoteAddr, region)),
				Level:     3,
				Status:    http.StatusUnauthorized,
			}
		}

		h := RandomString(32)

		// TODO: Set correct expiry time here
		err = rc.rclient.Set(rc.deviceid, h, 0).Err()
		if err != nil {
			return &HTTPError{
				IError:    err,
				Level:     3,
				Error:     "error in registering device",
				ErrorCode: ErrInternal,
				Status:    http.StatusInternalServerError,
			}
		}

		resp := &OkResponse{
			Status: http.StatusText(http.StatusOK),
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return handleJSONError(err)
		}
		return nil
	}
}

// VerifyDevice verifies an existing deviceid
//
// Input: Location, Device ID(deviceid), Hash(hash) in Query Parameters
//
// Output: Hash
func VerifyDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		h := r.URL.Query().Get("hash")
		if h == "" {
			return &HTTPError{
				ErrorCode: ErrNotFound,
				Level:     1,
				Status:    http.StatusBadRequest,
				Error:     "No Hash found",
			}
		}

		res, err := rc.rclient.Get(rc.deviceid).Result()
		if err != nil {
			if err == redis.Nil {
				return &HTTPError{
					Status:    http.StatusOK,
					ErrorCode: ErrNotRegistered,
					Error:     "Device is not registered",
					Level:     1,
				}
			}
			return &HTTPError{
				IError:    err,
				Level:     3,
				ErrorCode: ErrInternal,
				Error:     http.StatusText(http.StatusInternalServerError),
				Status:    http.StatusInternalServerError,
			}
		}

		if res != h {
			return &HTTPError{
				Level:     3,
				IError:    errors.New("hash mismatch"),
				Error:     "Hashes do not match",
				ErrorCode: ErrMismatch,
				Status:    http.StatusOK,
			}
		}

		resp := &OkResponse{
			Status: http.StatusText(http.StatusOK),
		}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return handleJSONError(err)
		}

		return nil
	}
}

// Fetch Latest, Fetch After Id, Serves Post, Timestamp, liked
func FetchPost() {}

// IP Address, DeviceID, Post, time, POSTid; Response: Time, POSTid
func SubmitPost() {}

// Verify DeviceID, -> input:newPost, timestamp; OK, timestamp
func EditPost() {}

// input: postid, devicehash; output: Total likes
func LikePost() {}

// input: postid, devicehash, output; total likes.
func dislikepost() {}

// input: Postid, output: Comments object array, Comment, Timestamp,
func fetchComments() {}

func submitComments() {}

// postid, deviceid, reason
func report() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		postid := r.Form.Get("postid")
		if postid == "" {
			return &HTTPError{
				Level:     1,
				Status:    http.StatusBadRequest,
				Error:     "postid not found",
				ErrorCode: ErrNotFound,
			}
		}

		reason := r.Form.Get("reason")
		if reason == "" {
			return &HTTPError{
				Level:     1,
				Status:    http.StatusBadRequest,
				Error:     "reason not found",
				ErrorCode: ErrNotFound,
			}
		}

		_, err := rc.pqdb.FetchPost(postid)
		if err != nil {
			return &HTTPError{
				Level:     3,
				Status:    http.StatusBadRequest,
				Error:     "invalid postid",
				ErrorCode: ErrNotFound,
				IError:    err,
			}
		}

		err = rc.pqdb.Report(postid, rc.deviceid, reason)
		if err != nil {
			return &HTTPError{
				IError:    err,
				ErrorCode: ErrInternal,
				Status:    http.StatusInternalServerError,
				Level:     3,
				Error:     "Error in reporting this post, Please retry in some time",
			}
		}

		resp := OkResponse{
			Status: http.StatusText(http.StatusOK),
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return handleJSONError(err)
		}
		return nil
	}
}

func handleJSONError(err error) *HTTPError {
	return &HTTPError{
		Error:     http.StatusText(http.StatusInternalServerError),
		ErrorCode: ErrInternal,
		IError:    err,
		Level:     3,
		Status:    http.StatusInternalServerError,
	}
}

// input: postid; output: post, timestamp, likes, comments
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
