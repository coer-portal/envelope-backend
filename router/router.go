package router

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ishanjain28/envelope-backend/common"
	"github.com/ishanjain28/envelope-backend/db"
	"github.com/ishanjain28/envelope-backend/log"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (

	// ErrInvalidNotFound is sent when an expected resource in request is unavailable
	ErrInvalidNotFound  = "INVALID_OR_NOT_FOUND"
	ErrNotRegistered    = "NOT_REGISTERED"
	ErrInternal         = "INTERNAL_ERROR"
	ErrOutOfValidRegion = "OUT_OF_REGION"
	ErrMismatch         = "MISMATCH"
	ErrParsing          = "PARSING_ERROR"
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
	deviceid  string `json:"-"`
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
					if e.deviceid != "" {
						log.Warn.Printf("[%s] %v: %s\n", e.deviceid, e.IError, e.IError)
					} else {
						log.Warn.Printf("%v: %s\n", e.IError, e.IError)
					}

				case 3:
					if e.deviceid != "" {
						log.Error.Printf("[%s] %v: %s\n", e.deviceid, e.IError, e.IError)
					} else {
						log.Warn.Printf("%v: %s\n", e.IError, e.IError)
					}
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
		registerDevice(),
	)).Methods("POST")

	r.Handle("/verify-device", Handle(client, nil,
		parseDeviceID(),
		verifyDevice(),
	)).Methods("GET")

	r.Handle("/report", Handle(client, pqdb,
		parseForm(),
		parseDeviceID(),
		report(),
	)).Methods("POST")

	r.Handle("/submit-post", Handle(client, pqdb,
		parseForm(),
		parseDeviceID(),
		submitPost(),
	)).Methods("POST")

	r.Handle("/fetch/{tag}", Handle(client, pqdb,
		fetchPost(),
	)).Methods("GET")

	r.Handle("/like-post", Handle(client, pqdb,
		parseForm(),
		parseDeviceID(),
		likePost(),
	)).Methods("POST")
	return r
}

// RegisterDevice receives a deviceid via POST and puts it in redis for 2 months, And sends a Hash back in response
func registerDevice() Handler {
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
				deviceid:  rc.deviceid,
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
				deviceid:  rc.deviceid,
				Error:     "error in registering device",
				ErrorCode: ErrInternal,
				Status:    http.StatusInternalServerError,
			}
		}

		resp := &RegisterDeviceResponse{
			Status: OK,
			Hash:   h,
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
// Output: {"Status": "OK"}
func verifyDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		h := r.URL.Query().Get("hash")
		if h == "" {
			return handleMissingDataError("hash")
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
				deviceid:  rc.deviceid,
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
			Status: OK,
		}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return handleJSONError(err)
		}

		return nil
	}
}

// Fetch Latest, Fetch After Id, Serves Post, Timestamp, liked
func fetchPost() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {
		v := mux.Vars(r)

		l := r.URL.Query().Get("limit")
		limit, err := strconv.Atoi(l)
		if limit == 0 || err != nil {
			limit = 20
		}

		tag := v["tag"]

		// Send Latest Posts
		if tag == "latest" {
			res, err := rc.pqdb.FetchNPosts(limit)
			if err != nil {
				return &HTTPError{
					Level:     3,
					deviceid:  rc.deviceid,
					Status:    http.StatusInternalServerError,
					Error:     "error in fetching latest posts",
					ErrorCode: ErrInternal,
					IError:    err,
				}
			}

			err = json.NewEncoder(w).Encode(res)
			if err != nil {
				return handleJSONError(err)
			}

			return nil
		}

		tagInt, err := strconv.Atoi(tag)
		if err != nil {
			return handleMissingDataError("postid")
		}

		prop := r.URL.Query().Get("prop")
		if prop == "" || (prop != "before" && prop != "after") {
			return handleMissingDataError("prop")
		}

		// Fetch Posts before specified id or after specified id
		// Send posts that were created after the specified postid
		// if the postid is invalid, Send a "Bad Request" Response
		res, err := rc.pqdb.FetchPostsFromID(tagInt, limit, prop)
		if err != nil {
			return &HTTPError{
				Level:     3,
				deviceid:  rc.deviceid,
				Error:     "error in fetching posts",
				ErrorCode: ErrInternal,
				IError:    err,
				Status:    http.StatusInternalServerError,
			}
		}

		err = json.NewEncoder(w).Encode(res)
		if err != nil {
			return handleJSONError(err)
		}
		return nil
	}
}

// IP Address, DeviceID, Post, time, POSTid; Response: Time, POSTid
func submitPost() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		post := r.Form.Get("post")
		if post == "" {
			return handleMissingDataError("post")
		}

		timestamp := time.Now().Unix()

		p := &db.Post{
			DeviceID:  rc.deviceid,
			Timestamp: timestamp,
			Text:      post,
			IPAddr:    fetchRemoteIpAddr(r.RemoteAddr),
		}

		err := rc.pqdb.SubmitPost(p)

		if err != nil {
			return &HTTPError{
				Level:     3,
				deviceid:  rc.deviceid,
				IError:    err,
				Error:     "error in saving Post, Please retry",
				Status:    http.StatusInternalServerError,
				ErrorCode: ErrInternal,
			}
		}

		// p.ID is set in SubmitPost after retrieving ID of post inserted in database
		resp := &SubmitPostResponse{
			likes:     0,
			PostID:    p.ID,
			Status:    OK,
			Timestamp: timestamp,
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return handleJSONError(err)
		}

		return nil
	}
}

// Verify DeviceID, -> input:newPost, timestamp; OK, timestamp
func editPost() {}

// input: postid, devicehash; output: Total likes
func likePost() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		postid := r.Form.Get("postid")
		if postid == "" {
			return handleMissingDataError("postid")
		}

		err := rc.pqdb.LikePost(postid, rc.deviceid)
		if err != nil {

			if err.Error() == db.ErrInvalidPostID {

				return &HTTPError{
					Error:     "invalid postid",
					ErrorCode: ErrInvalidNotFound,
					Level:     1,
					Status:    http.StatusBadRequest,
				}
			}

			return &HTTPError{
				Level:    3,
				Status:   http.StatusInternalServerError,
				IError:   err,
				deviceid: rc.deviceid,
			}
		}

		ok := &OkResponse{Status: OK}

		err = json.NewEncoder(w).Encode(ok)
		if err != nil {
			return handleJSONError(err)
		}

		return nil
	}
}

// input: Postid, output: Comments object array, Comment, Timestamp,
func fetchComments() {}

func submitComment() {}

// postid, deviceid, reason
func report() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		postid := r.Form.Get("postid")
		if postid == "" {
			return handleMissingDataError("postid")
		}

		reason := r.Form.Get("reason")
		if reason == "" {
			return handleMissingDataError("reason")
		}

		err := rc.pqdb.Report(postid, rc.deviceid, reason)
		if err != nil {

			if err == sql.ErrNoRows {
				return handleMissingDataError("postid")
			}

			return &HTTPError{
				IError:    err,
				ErrorCode: ErrInternal,
				deviceid:  rc.deviceid,
				Status:    http.StatusInternalServerError,
				Level:     3,
				Error:     "Error in reporting this post, Please retry in some time",
			}
		}

		resp := OkResponse{
			Status: OK,
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return handleJSONError(err)
		}
		return nil
	}
}
