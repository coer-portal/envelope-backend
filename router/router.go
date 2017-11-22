package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ishanjain28/envelope-backend/common"
	"github.com/ishanjain28/envelope-backend/db"
	"github.com/ishanjain28/envelope-backend/log"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (
	workingRegion = "Uttarakhand"
)

type RouterContext struct {
	db       db.IDB
	deviceid string
	ctx      context.Context
}

type HTTPError struct {
	Level     int    `json:"-"`
	IError    error  `json:"-"`
	Status    int    `json:"status"`
	ErrorCode string `json:"error_code"`
	deviceid  string `json:"-"`
}

func (e HTTPError) Error() string {
	return e.IError.Error()
}

type Handler func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError

func Handle(pqre db.IDB, handlers ...Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// context for redis is set here and provided with first arguments in relevant functions
		//pqre.Redis = pqre.Redis.WithContext(ctx)

		rc := &RouterContext{
			db:  pqre,
			ctx: ctx,
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
						log.Warn.Printf("[%s] %s\n", e.deviceid, e.IError)
					} else {
						log.Warn.Printf("%s\n", e.IError, e.IError)
					}

				case 3:
					if e.deviceid != "" {
						log.Error.Printf("[%s] %s\n", e.deviceid, e.IError)
					} else {
						log.Warn.Printf("%s\n", e.IError)
					}

					if e.IError == context.DeadlineExceeded {
						e.Status = http.StatusRequestTimeout
						e.ErrorCode = ErrTimeout
					}

					w.WriteHeader(e.Status)
					err := json.NewEncoder(w).Encode(e)
					if err != nil {
						log.Error.Printf("%s\n", err, err)
						w.Header().Set("Content-Type", "text/plain")
						w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
					}
					return
				}
			}
		}
	})
}

func Init(pqre db.IDB) *mux.Router {
	r := mux.NewRouter()

	r.Handle("/register-device", Handle(pqre,
		parseDeviceID(),
		registerDevice(),
		parseForm(),
	)).Methods("POST")

	r.Handle("/verify-device", Handle(pqre,
		parseDeviceID(),
		verifyDevice(),
	)).Methods("GET")

	r.Handle("/report", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
		parseForm(),
		report(),
	)).Methods("POST")

	r.Handle("/submit-post", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
		parseForm(),
		submitPost(),
	)).Methods("POST")

	r.Handle("/edit-post", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
		parseForm(),
		editPost(),
	)).Methods("POST")

	r.Handle("/fetch/{tag}", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
		fetchPost(),
	)).Methods("GET")

	r.Handle("/like-post", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
		parseForm(),
		likePost(),
	)).Methods("POST")

	r.Handle("/comment", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
		parseForm(),
	)).Methods("POST")

	r.Handle("/fetch-comments", Handle(pqre,
		parseDeviceID(),
		verifyDeviceID(),
	)).Methods("GET")

	return r
}

// RegisterDevice receives a deviceid via POST and puts it in redis for 2 months, And sends a Hash back in response
func registerDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		// TODO: Add Context
		region, err := common.GetRegionofIP(common.GetIPAddr(r))
		if err != nil {
			return &HTTPError{
				ErrorCode: ErrInternal,
				Level:     3,
				Status:    http.StatusInternalServerError,
				IError:    err,
			}
		}

		if region != workingRegion {
			return &HTTPError{
				ErrorCode: ErrOutOfValidRegion,
				deviceid:  rc.deviceid,
				IError:    errors.New(fmt.Sprintf("%s: %s is from %s", ErrOutOfValidRegion, common.GetIPAddr(r), region)),
				Level:     3,
				Status:    http.StatusUnauthorized,
			}
		}

		h := RandomString(20)

		// TODO: Set correct expiry time here
		err = rc.db.RegisterDeviceID(rc.ctx, rc.deviceid, h, 0)
		if err != nil {
			return &HTTPError{
				IError:    err,
				Level:     3,
				deviceid:  rc.deviceid,
				ErrorCode: ErrInternal,
				Status:    http.StatusInternalServerError,
			}
		}

		Send(&RegisterDeviceResponse{
			Status: OK,
			Hash:   h,
		}, w)
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

		hash, err := rc.db.VerifyDeviceID(rc.ctx, rc.deviceid)
		if err != nil {

			if err.Error() == ErrNotRegistered {
				return &HTTPError{
					ErrorCode: ErrNotRegistered,
					Status:    http.StatusOK,
					Level:     1,
				}
			}

			return &HTTPError{
				deviceid:  rc.deviceid,
				ErrorCode: ErrInternal,
				IError:    err,
				Level:     3,
				Status:    http.StatusInternalServerError,
			}
		}

		if hash != h {
			return &HTTPError{
				ErrorCode: ErrExpired,
				Status:    http.StatusOK,
				Level:     1,
			}
		}

		Send(&OkResponse{Status: OK}, w)

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

		posts := []*db.Post{}

		// Send Latest Posts
		if tag == "latest" {
			posts, err = rc.db.FetchNPosts(rc.ctx, limit)
			if err != nil {
				return &HTTPError{
					Level:     3,
					deviceid:  rc.deviceid,
					Status:    http.StatusInternalServerError,
					ErrorCode: ErrInternal,
					IError:    err,
				}
			}
		} else {

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
			posts, err = rc.db.FetchPostsFromID(rc.ctx, tagInt, limit, prop)
			if err != nil {
				return &HTTPError{
					Level:     3,
					deviceid:  rc.deviceid,
					ErrorCode: ErrInternal,
					IError:    err,
					Status:    http.StatusInternalServerError,
				}
			}
		}
		Send(posts, w)

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
			IPAddr:    fetchRemoteIpAddr(common.GetIPAddr(r)),
		}

		err := rc.db.SubmitPost(rc.ctx, p)

		if err != nil {
			return &HTTPError{
				Level:     3,
				deviceid:  rc.deviceid,
				IError:    err,
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

		Send(resp, w)
		return nil
	}
}

// Verify DeviceID, -> input:newPost, timestamp; OK, timestamp
func editPost() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		post := r.Form.Get("post")
		if post == "" {
			return handleMissingDataError("post")
		}

		postid := r.Form.Get("postid")
		if postid == "" {
			return handleMissingDataError("postid")
		}

		//timestamp := time.Now().Unix()
		//p, err := rc.db.FetchPost(rc.ctx, postid)

		return nil
	}
}

// input: postid, devicehash; output: Total likes
func likePost() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		postid := r.Form.Get("postid")
		if postid == "" {
			return handleMissingDataError("postid")
		}

		err := rc.db.LikePost(rc.ctx, postid, rc.deviceid)
		if err != nil {

			if err.Error() == db.ErrInvalidPostID {

				return &HTTPError{
					ErrorCode: ErrInvalidData,
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

		Send(ok, w)

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

		err := rc.db.Report(rc.ctx, postid, rc.deviceid, reason)
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
			}
		}

		resp := OkResponse{
			Status: OK,
		}

		Send(resp, w)

		return nil
	}
}

func Send(v interface{}, w http.ResponseWriter) *HTTPError {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return handleJSONError(err)
	}

	return nil
}
