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

	pqdb := &DB{Db: db}

	log.Info.Println("Creating Tables")

	log.Info.Println("Creating reports table")
	err = pqdb.createTables("CREATE TABLE reports(id SERIAL PRIMARY KEY, postid VARCHAR NOT NULL, deviceid VARCHAR NOT NULL, reason VARCHAR NOT NULL)")
	if err != nil {
		return nil, err
	}

	log.Info.Println("Creating posts table")
	err = pqdb.createTables("CREATE TABLE posts (postid VARCHAR PRIMARY KEY, deviceid VARCHAR NOT NULL, post VARCHAR NOT NULL, likes INTEGER NOT NULL, dislikes INTEGER NOT NULL, timestamp INTEGER NOT NULL, ipaddr CIDR NOT NULL)")
	if err != nil {
		return nil, err
	}
	log.Info.Println("Created posts table")

	log.Info.Printf("Tables Created...")
	return &DB{Db: db}, nil
}

func (d *DB) FetchNPosts(n int) {}

func (d *DB) FetchAfterID(id string) {}

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