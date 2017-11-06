package db

type Post struct {
	ID        int64  `db:"postid" json:"postid"`
	Text      string `db:"post" json:"post"`
	Timestamp int64  `db:"timestamp" json:"timestamp"`
	DeviceID  string `db:"deviceid" json:"-"`
	Comments  int    `db:"comments" json:"comments"`
	Likes     int    `db:"likes" json:"likes"`
	IPAddr    string `db:"ipAddr" json:"-"`
}
