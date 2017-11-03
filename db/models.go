package db

import "net"

type Post struct {
	ID        string  `db:"postid"`
	Text      string  `db:"post"`
	Timestamp int64   `db:"timestamp"`
	DeviceID  string  `db:"deviceid"`
	Likes     int     `db:"likes"`
	Dislikes  int     `db:"dislikes"`
	IPAddr    *net.IP `db:"ipaddr"`
}
