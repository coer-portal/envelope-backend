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
	ErrNotFound         = "NOTFOUND"
	ErrNotExists        = "NOTEXISTS"
	ErrInternal         = "INTERNALERROR"
	ErrOutOfValidRegion = "OUTOFREGION"
	workingRegion       = "Uttarakhand"
)

type RouterContext struct {
	rclient *redis.Client
	pqdb    *db.DB
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
		RegisterDevice(),
	)).Methods("POST")

	r.Handle("/verify-device", Handle(client, nil,
		VerifyDevice(),
	)).Methods("GET")
	return r
}

// RegisterDevice receives a deviceid via POST and puts it in redis for 2 months, And sends a Hash back in response
func RegisterDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		err := r.ParseForm()
		if err != nil {
			return &HTTPError{
				IError: err, Level: 1, Error: "error in parsing form", Status: http.StatusBadRequest}
		}

		deviceid := r.Form.Get("deviceid")
		if deviceid == "" {
			return &HTTPError{
				IError:    errors.New("No Device Id"),
				Level:     1,
				Error:     "Nr Device Id",
				ErrorCode: ErrNotFound,
				Status:    http.StatusBadRequest,
			}
		}

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
				Level:     1,
				Status:    http.StatusUnauthorized,
			}
		}

		h := RandomString(32)

		err = rc.rclient.Set(deviceid, h, 0).Err()
		if err != nil {
			return &HTTPError{
				IError:    err,
				Level:     3,
				Error:     "error in registering device",
				ErrorCode: ErrInternal,
				Status:    http.StatusInternalServerError,
			}
		}

		resp := &RegisterDeviceResponse{
			Hash: h,
		}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return &HTTPError{
				IError:    err,
				Level:     3,
				Error:     http.StatusText(http.StatusInternalServerError),
				ErrorCode: ErrInternal,
				Status:    http.StatusInternalServerError,
			}
		}
		return nil
	}
}

// Location, Input -> Deviceid, Hash
func VerifyDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		deviceid := r.URL.Query().Get("deviceid")
		if deviceid == "" {
			return &HTTPError{
				IError:    errors.New("No device id"),
				Level:     1,
				ErrorCode: ErrNotFound,
				Error:     "No Device id",
				Status:    http.StatusBadRequest,
			}
		}
		res, err := rc.rclient.Get(deviceid).Result()
		if err != nil {
			if err == redis.Nil {
				return &HTTPError{
					IError:    errors.New("No Results Found"),
					Status:    http.StatusOK,
					ErrorCode: ErrNotExists,
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

		resp := &RegisterDeviceResponse{
			Hash: res,
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return &HTTPError{
				Error:     http.StatusText(http.StatusInternalServerError),
				ErrorCode: ErrInternal,
				IError:    err,
				Level:     3,
				Status:    http.StatusInternalServerError,
			}
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
func report() {}

// input: postid; output: post, timestamp, likes, comments

func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
