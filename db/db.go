package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/envelope-app/envelope-backend/log"
	"github.com/go-redis/redis"
	"github.com/lib/pq"
)

type DB struct {
	Pq    *sql.DB
	Redis *redis.Client
}

// IDB interface defines all the database operations used by the application.
type IDB interface {
	FetchNPosts(ctx context.Context, n int) ([]*Post, error)
	FetchPostsFromID(ctx context.Context, id, limit int, prop string) ([]*Post, error)
	LikePost(ctx context.Context, postid, deviceid string) error
	Report(ctx context.Context, postid, deviceid, reason string) error
	SubmitPost(ctx context.Context, p *Post) error
	//Comment(ctx context.Context, postid string, comment *Comment) error
	//FetchPostComments(ctx context.Context, postid string) ([]*Comment, error)

	// Authentication related endpoints
	VerifyDeviceID(ctx context.Context, deviceid string) (string, error)
	RegisterDeviceID(ctx context.Context, deviceid, hash string, t time.Duration) error
}

var (
	postgresAddr = os.Getenv("DATABASE_URL")
	redisAddr    = os.Getenv("REDISTOGO_URL")

	ErrInvalidPostID = "INVALID_POST_ID"
	ErrAlreadyLiked  = "ALREADY_LIKED"
)

func Init() (IDB, error) {

	if postgresAddr == "" {
		return nil, errors.New("$POSTGRES_URL not set")
	}

	if redisAddr == "" {
		return nil, errors.New("$REDIS_SERVER not set")
	}

	// Connect to Postgresql
	pq, err := sql.Open("postgres", postgresAddr)
	if err != nil {
		return nil, err
	}

	// Parse REDIS_SERVER and connect to Redis Server
	redisOpt, err := redis.ParseURL(redisAddr)
	if err != nil {
		log.Error.Fatalf("Invalid $REDIS_SERVER: %v\n", err)
	}

	client := redis.NewClient(redisOpt)

	err = client.Ping().Err()
	if err != nil {
		log.Error.Fatalf("Error in connecting to redis: %s", err)
	}

	db := &DB{Pq: pq, Redis: client}

	// Initialize tables befor returning
	err = db.createTables()
	if err != nil {
		return nil, err
	}
	return IDB(db), nil
}

// SubmitPost takes a Post, puts it into the database and returns the postid
func (d *DB) SubmitPost(ctx context.Context, p *Post) error {

	var id int
	query := fmt.Sprintf("INSERT INTO posts(deviceid, post, timestamp, ipaddr) VALUES ('%s', '%s', '%d', '%s') RETURNING postid", p.DeviceID, p.Text, p.Timestamp, p.IPAddr)

	err := d.Pq.QueryRowContext(ctx, query).Scan(&id)
	if err != nil {
		return err
	}

	log.Info.Printf("saved 1 post(%d) from %s\n", id, p.DeviceID)

	//TODO: Consider returning postid instead of mutating Post
	p.ID = id
	return nil
}

// FetchNPosts takes an integer and returns the most recent N posts
func (d *DB) FetchNPosts(ctx context.Context, n int) ([]*Post, error) {
	query := fmt.Sprintf("SELECT postid, deviceid, post, timestamp FROM posts ORDER BY postid DESC LIMIT %d", n)

	rows, err := d.Pq.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	p := []*Post{}

	for rows.Next() {
		post := &Post{}

		err = rows.Scan(&post.ID, &post.DeviceID, &post.Text, &post.Timestamp)
		if err != nil {
			return nil, err
		}
		p = append(p, post)
	}

	return p, nil
}

// FetchPostsFromID fetches a number of posts before or after the specified id
func (d *DB) FetchPostsFromID(ctx context.Context, id, limit int, prop string) ([]*Post, error) {

	timestampquery := fmt.Sprintf("SELECT timestamp FROM posts WHERE postid='%d'", id)

	row := d.Pq.QueryRowContext(ctx, timestampquery)
	var t int64

	err := row.Scan(&t)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	var query string
	if prop == "after" {
		// Select N posts newer than the specified post and include the specified post.
		query = fmt.Sprintf("SELECT postid, deviceid, post, timestamp FROM posts WHERE timestamp >= %d AND postid >= %d LIMIT %d", t, id, limit)
	} else {
		// Select N posts older than the specified post and exclude the specified post.
		query = fmt.Sprintf("SELECT postid, deviceid, post, timestamp FROM posts WHERE timestamp < %d AND postid <= %d ORDER BY postid DESC LIMIT %d", t, id, limit)
	}
	rows, err := d.Pq.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	p := []*Post{}

	for rows.Next() {
		post := &Post{}

		err := rows.Scan(&post.ID, &post.DeviceID, &post.Text, &post.Timestamp)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		p = append(p, post)
	}

	return p, nil
}

// Report puts information like postid and device id in reports table
func (d *DB) Report(ctx context.Context, postid, deviceid, reason string) error {

	postExistsQuery := fmt.Sprintf("SELECT postid FROM posts where postid='%s'", postid)

	pid := 0
	// Verify that the specified postid exists
	err := d.Pq.QueryRowContext(ctx, postExistsQuery).Scan(&pid)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("INSERT INTO reports(postid, deviceid, reason) VALUES ('%s', '%s', '%s')", postid, deviceid, reason)

	_, err = d.Pq.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	log.Info.Printf("Saved report for %s from %s", postid, deviceid)

	return nil
}

// LikePost adds a new entry in likes table containing details like deviceid and postid.
// When complete details of a post are requested, We'll have to count all the entries containing specified postid.
func (d *DB) LikePost(ctx context.Context, postid string, deviceid string) error {

	query := fmt.Sprintf("INSERT INTO likes(postid, deviceid) VALUES('%s', '%s')", postid, deviceid)

	p, err := d.FetchPost(ctx, postid)
	if err != nil {
		return err
	}

	if p == nil {
		return errors.New(ErrInvalidPostID)
	}

	_, err = d.Pq.ExecContext(ctx, query)
	if err != nil {
		log.Warn.Println(err.(pq.Error).Code.Name())
		return err
	}

	return nil
}

func (d *DB) fetchComments(ctx context.Context, postid string) ([]*Comment, error) {
	query := fmt.Sprintf("SELECT commentid, comment, timestamp FROM comments WHERE postid='%s'", postid)

	c := []*Comment{}

	rows, err := d.Pq.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		co := &Comment{}

		err = rows.Scan(&co.ID, &co.Text, &co.Timestamp)

		if err != nil {
			return nil, err
		}

		c = append(c, co)
	}

	return c, nil
}

func (d *DB) fetchLikes(ctx context.Context, postid string) (int, error) {
	query := fmt.Sprintf("SELECT count(*) FROM likes WHERE postid='%s'", postid)

	likes := 0

	err := d.Pq.QueryRowContext(ctx, query).Scan(&likes)
	if err != nil {
		return 0, err
	}

	return likes, nil
}

func (d *DB) FetchPost(ctx context.Context, postid string) (*Post, error) {

	query := fmt.Sprintf("SELECT postid, deviceid, post, timestamp, ipaddr FROM posts WHERE postid='%s'", postid)

	p := &Post{}

	err := d.Pq.QueryRowContext(ctx, query).Scan(&p.ID, &p.DeviceID, &p.Text, &p.Timestamp, &p.IPAddr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return p, nil
}

func (d *DB) createTables() error {

	log.Info.Println("Creating Tables")

	log.Info.Println("Creating reports table")
	err := d.createTableHelper("CREATE TABLE reports(reportid SERIAL PRIMARY KEY, postid INTEGER NOT NULL, deviceid VARCHAR NOT NULL, reason VARCHAR NOT NULL)")
	if err != nil {
		return err
	}

	log.Info.Println("Creating posts table")
	err = d.createTableHelper("CREATE TABLE posts (postid SERIAL PRIMARY KEY, deviceid VARCHAR NOT NULL, post VARCHAR NOT NULL, timestamp INTEGER NOT NULL, ipaddr VARCHAR NOT NULL)")
	if err != nil {
		return err
	}
	log.Info.Println("Created posts table")

	log.Info.Println("Creating Comments Table")
	err = d.createTableHelper("CREATE TABLE comments(commentid SERIAL PRIMARY KEY, postid INTEGER NOT NULL, deviceid VARCHAR NOT NULL, timestamp INTEGER NOT NULL, comment VARCHAR NOT NULL)")
	if err != nil {
		return err
	}
	log.Info.Println("Created Comments Table")

	log.Info.Println("Creating Likes Table")
	err = d.createTableHelper("CREATE TABLE likes(postid INTEGER NOT NULL, deviceid VARCHAR NOT NULL, PRIMARY KEY(postid, deviceid))")
	if err != nil {
		return err
	}
	log.Info.Println("Created Likes Table")

	log.Info.Printf("Tables Created...")

	return nil
}

func (d *DB) createTableHelper(stmt string) error {
	_, err := d.Pq.Exec(stmt)
	if err != nil {
		if perr, ok := err.(*pq.Error); ok {
			if perr.Code.Name() != "duplicate_table" {
				return perr
			}
			log.Warn.Printf("%s: %s", perr.Code.Name(), perr.Error())
			return nil
		}
		return err
	}
	return nil
}
