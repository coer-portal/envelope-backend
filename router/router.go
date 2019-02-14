package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/envelope-app/envelope-backend/envelope"

	"github.com/envelope-app/envelope-backend/db"
	"github.com/envelope-app/envelope-backend/log"
	"github.com/gorilla/mux"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (
	workingRegion = "Uttarakhand"
)

type Router struct {
}

// RouterContext holds all the connections/information a request will need
type RouterContext struct {
	db       db.IDB
	deviceid string
	ctx      context.Context
}

// Handler interface provides for a easy, convenient middleware pattern
type Handler func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError

// Handle executes all the Handlers one by one.
// Error from an handler is evaluated after it's execution and depending on the level of error
// The decision to execute next middleware is taken.
// Currently, There are 3 levels of errors.
// Level 1,2 and 3. i
// Level 1 errors are Bad requests, Or anything that is just the fault of user and there is no advantage in logging them
// Level 2 errors are warnings, Something that might be important to the server. These errors are logged to console but the request is moved forward to next middleware.
// Level 3, All hell broke loose, Log the request and send an appropriate error response to the user, Don't forward request to next middleware
func Handle(pqre db.IDB, handlers ...Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
				switch e.Level {
				case 1:
					w.WriteHeader(e.Code)
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
						log.Warn.Println(e.IError)
					}

				case 3:
					if e.deviceid != "" {
						log.Error.Printf("[%s] %s\n", e.deviceid, e.IError)
					} else {
						log.Warn.Println(e.IError)
					}

					if e.IError == context.DeadlineExceeded {
						e.Code = http.StatusRequestTimeout
						e.ErrorCode = ErrTimeout
					}

					w.WriteHeader(e.Code)
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

func (r *Router) RegisterDevice(ctx context.Context, input *envelope.RegisterDeviceInput) (*envelope.RegisterDeviceOutput, error) {
	if input.Deviceid != "" {
		return  &envelope.RegisterDeviceOutput{
			Code: 400,
		}, nil
	}

	err = rc.db.RegisterDeviceID(rc.ctx, rc.deviceid, 0)
	if err != nil {
		return &envelope.RegisterDeviceOutput {
			Code: 500,
		}, nil
	}


	return 	&envelope.RegisterDeviceOutput{
		Code: 200,
	}, w), nil
}

func Init(pqre db.IDB) Router {
r := Router{}

	// r.Handle("/verify-device", Handle(pqre,
	// 	verifyDevice(),
	// )).Methods("GET")

	// r.Handle("/report", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// 	parseForm(),
	// 	report(),
	// )).Methods("POST")

	// r.Handle("/submit-post", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// 	parseForm(),
	// 	submitPost(),
	// )).Methods("POST")

	// r.Handle("/edit-post", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// 	parseForm(),
	// 	editPost(),
	// )).Methods("POST")

	// r.Handle("/fetch/{tag}", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// 	fetchPost(),
	// )).Methods("GET")

	// r.Handle("/like-post", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// 	parseForm(),
	// 	likePost(),
	// )).Methods("POST")

	// r.Handle("/comment", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// 	parseForm(),
	// )).Methods("POST")

	// r.Handle("/fetch-comments", Handle(pqre,
	// 	parseDeviceID(),
	// 	verifyDeviceID(),
	// )).Methods("GET")

	return r
}

// VerifyDevice verifies an existing deviceid
//
// Input: Location, Device ID(deviceid), Hash(hash) in Query Parameters
//
// Output: {"Status": "OK"}
// Don't need hash
func verifyDevice() Handler {
	return func(rc *RouterContext, w http.ResponseWriter, r *http.Request) *HTTPError {

		h := r.Header.Get("hash")
		if h == "" {
			return handleMissingDataError("hash")
		}

		hash, err := rc.db.VerifyDeviceID(rc.ctx, rc.deviceid)
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

		if hash != h {
			return &HTTPError{
				ErrorCode:       ErrExpired,
				GenericResponse: HTTPResponse(http.StatusBadRequest),
				Level:           1,
			}
		}

		Send(HTTPResponse(http.StatusOK), w)

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
					Level:           3,
					deviceid:        rc.deviceid,
					GenericResponse: HTTPResponse(http.StatusInternalServerError),
					ErrorCode:       ErrInternal,
					IError:          err,
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
					Level:           3,
					deviceid:        rc.deviceid,
					ErrorCode:       ErrInternal,
					IError:          err,
					GenericResponse: HTTPResponse(http.StatusInternalServerError),
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
		}

		err := rc.db.SubmitPost(rc.ctx, p)

		if err != nil {
			return &HTTPError{
				Level:           3,
				deviceid:        rc.deviceid,
				IError:          err,
				GenericResponse: HTTPResponse(http.StatusInternalServerError),
				ErrorCode:       ErrInternal,
			}
		}

		// p.ID is set in SubmitPost after retrieving ID of post inserted in database
		resp := &SubmitPostResponse{
			likes:           0,
			PostID:          p.ID,
			Timestamp:       timestamp,
			GenericResponse: HTTPResponse(http.StatusOK),
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
					ErrorCode:       ErrInvalidData,
					Level:           1,
					GenericResponse: HTTPResponse(http.StatusBadRequest),
				}
			}

			return &HTTPError{
				Level:           3,
				GenericResponse: HTTPResponse(http.StatusInternalServerError),
				IError:          err,
				deviceid:        rc.deviceid,
			}
		}

		Send(HTTPResponse(http.StatusOK), w)

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
				IError:          err,
				ErrorCode:       ErrInternal,
				deviceid:        rc.deviceid,
				GenericResponse: HTTPResponse(http.StatusInternalServerError),
				Level:           3,
			}
		}

		Send(HTTPResponse(http.StatusOK), w)

		return nil
	}
}
