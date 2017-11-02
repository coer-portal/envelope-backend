package router

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ishanjain28/envelope-backend/db"
	"github.com/ishanjain28/envelope-backend/log"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type RouterContext struct {
	rclient *redis.Client
	pqdb    *db.DB
}

type HTTPError struct {
	Level   int
	Error   error
	Status  int
	Message string
}

type Handler func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError

func Handle(client *redis.Client, pqdb *db.DB, handlers ...Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rc := &RouterContext{
			rclient: client,
			pqdb:    pqdb,
		}

		for _, handler := range handlers {
			e := handler(rc, w, r)
			if e != nil {
				if e.Level < 2 {
					log.Warn.Printf("%v: %s\n", e)
				} else {
					log.Error.Printf("%s: %s\n", e.Message, e.Error.Error())
					w.WriteHeader(e.Status)
					w.Write([]byte(e.Message))
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

	r.Handle("/verify-device", Handle(client, nil)).Methods("GET")
	return r
}

// RegisterDevice receives a deviceid via POST and puts it in redis for 2 months, And sends a Hash back in response
func RegisterDevice() Handler {

	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		err := r.ParseForm()
		if err != nil {
			return &HTTPError{
				Error: err, Level: 2, Message: "Error in parsing form", Status: http.StatusBadRequest}
		}

		deviceid := r.Form.Get("deviceid")
		if deviceid == "" {
			return &HTTPError{
				Error:   errors.New("No Device Id"),
				Level:   2,
				Message: "No Device Id",
				Status:  http.StatusBadRequest,
			}
		}

		h := RandomString(32)

		err = rc.rclient.Set(deviceid, h, 2*30*24*60*60).Err()
		if err != nil {
			return &HTTPError{
				Error:   err,
				Level:   2,
				Message: "Internal Server Error",
				Status:  http.StatusInternalServerError,
			}
		}

		resp := &RegisterDeviceResponse{
			Hash: h,
		}

		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return &HTTPError{
				Error:   err,
				Level:   2,
				Message: http.StatusText(http.StatusInternalServerError),
				Status:  http.StatusInternalServerError,
			}
		}
		return nil
	}
}

// Location, Input -> Deviceid, Hash
func VerifyDevice() {}

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
