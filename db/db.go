package db

import (
	"database/sql"
	"errors"
	"os"

	"github.com/ishanjain28/envelope-backend/log"
	"github.com/lib/pq"
)

type DB struct {
	Db *sql.DB
}

var postgresAddr = os.Getenv("POSTGRES_URL")

func Init() (*DB, error) {

	if postgresAddr == "" {
		return nil, errors.New("$POSTGRES_URL not set")
	}

	db, err := sql.Open("postgres", postgresAddr)
	if err != nil {
		return nil, err
	}

	log.Info.Printf("Creating Tables...\n")
	_, err = db.Exec("CREATE TABLE reports(id SERIAL PRIMARY KEY, postid VARCHAR NOT NULL, deviceid VARCHAR NOT NULL, reason VARCHAR NOT NULL)")
	if err != nil {
		if perr, ok := err.(*pq.Error); ok {

			if perr.Code.Name() != "duplicate_table" {
				return nil, perr
			} else {
				log.Warn.Printf("%s: %s", perr.Code.Name(), perr.Error())
			}
		} else {
			return nil, err
		}
	}

	log.Info.Printf("Tables Created...")
	return &DB{Db: db}, nil
}

func (d *DB) FetchNPosts() {}

func (d *DB) FetchAfterID() {}
