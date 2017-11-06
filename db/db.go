package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/ishanjain28/envelope-backend/log"
	"github.com/lib/pq"
)

type DB struct {
	Db *sql.DB
}

var (
	postgresAddr = os.Getenv("POSTGRES_URL")
	ErrNotFound  = "not found"
)

func Init() (*DB, error) {

	if postgresAddr == "" {
		return nil, errors.New("$POSTGRES_URL not set")
	}

	db, err := sql.Open("postgres", postgresAddr)
	if err != nil {
		return nil, err
	}

	pqdb := &DB{Db: db}

	log.Info.Println("Creating Tables")

	log.Info.Println("Creating reports table")
	err = pqdb.createTables("CREATE TABLE reports(reportid SERIAL PRIMARY KEY, postid VARCHAR NOT NULL, deviceid VARCHAR NOT NULL, reason VARCHAR NOT NULL)")
	if err != nil {
		return nil, err
	}

	log.Info.Println("Creating posts table")
	err = pqdb.createTables("CREATE TABLE posts (postid SERIAL PRIMARY KEY, deviceid VARCHAR NOT NULL, post VARCHAR NOT NULL, likes INTEGER NOT NULL, comments INTEGER NOT NULL, timestamp INTEGER NOT NULL, ipaddr VARCHAR NOT NULL)")
	if err != nil {
		return nil, err
	}
	log.Info.Println("Created posts table")

	log.Info.Println("Creating Comments Table")

	//	err := pqdb.createTables("CREATE TABLE comments(id SERIAL PRIMARY KEY, postid VARCHAR, deviceid VARCHAR NOT NULL,")

	log.Info.Println("Created Comments Table")

	log.Info.Printf("Tables Created...")
	return &DB{Db: db}, nil
}

func (d *DB) SubmitPost(p *Post) error {

	var id int64
	query := fmt.Sprintf("INSERT INTO posts(deviceid, post, likes, comments, timestamp, ipaddr) VALUES ('%s', '%s', '%d', '%d', '%d', '%s') RETURNING postid", p.DeviceID, p.Text, p.Likes, p.Comments, p.Timestamp, p.IPAddr)

	err := d.Db.QueryRow(query).Scan(&id)
	if err != nil {
		return err
	}

	log.Info.Printf("Saved 1 post(%d) from %s\n", id, p.DeviceID)

	p.ID = id
	return nil
}

func (d *DB) FetchNPosts(n int) ([]*Post, error) {
	query := fmt.Sprintf("SELECT postid, deviceid, post, likes, comments, timestamp FROM posts ORDER BY timestamp DESC LIMIT %d", n)

	rows, err := d.Db.Query(query)
	if err != nil {
		return nil, err
	}

	p := []*Post{}

	for rows.Next() {
		post := &Post{}

		err = rows.Scan(&post.ID, &post.DeviceID, &post.Text, &post.Likes, &post.Comments, &post.Timestamp)
		if err != nil {
			return nil, err
		}
		p = append(p, post)
	}

	return p, nil
}

// FetchPostsFromID fetches a number of posts before or after the specified id
func (d *DB) FetchPostsFromID(id int, limit int, prop string) ([]*Post, error) {

	timestampquery := fmt.Sprintf("SELECT timestamp FROM posts WHERE postid='%d'", id)

	row := d.Db.QueryRow(timestampquery)
	var t int64

	err := row.Scan(&t)
	if err != nil {
		return nil, err
	}

	var query string
	if prop == "after" {
		query = fmt.Sprintf("SELECT postid, deviceid, post, likes, comments, timestamp FROM posts WHERE timestamp >= %d LIMIT %d", t, limit)
	} else {
		query = fmt.Sprintf("SELECT postid, deviceid, post, likes, comments, timestamp FROM posts WHERE timestamp < %d ORDER BY timestamp DESC LIMIT %d", t, limit)
	}
	rows, err := d.Db.Query(query)
	if err != nil {
		return nil, err
	}

	p := []*Post{}

	for rows.Next() {
		post := &Post{}

		err := rows.Scan(&post.ID, &post.DeviceID, &post.Text, &post.Likes, &post.Comments, &post.Timestamp)
		if err != nil {
			return nil, err
		}
		p = append(p, post)
	}

	return p, nil
}

func (d *DB) Report(postid, deviceid, reason string) error {

	query := fmt.Sprintf("INSERT INTO reports(postid, deviceid, reason) VALUES ('%s', '%s', '%s')", postid, deviceid, reason)

	_, err := d.Db.Exec(query)
	if err != nil {
		return err
	}

	log.Info.Printf("Saved report for %s from %s", postid, deviceid)

	return nil
}

// FetchPost sends complete details of a post, Including all the comments on that post
// TODO: incomplete
func (d *DB) FetchPost(postid string) (*Post, error) {

	row := d.Db.QueryRow(fmt.Sprintf("SELECT  FROM posts WHERE postid='%s'", postid))

	p := &Post{}

	err := row.Scan()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (d *DB) createTables(stmt string) error {
	_, err := d.Db.Exec(stmt)
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
