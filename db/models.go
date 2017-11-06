package db

type Post struct {
	ID        int    `db:"postid" json:"postid"`
	Text      string `db:"post" json:"post"`
	Timestamp int64  `db:"timestamp" json:"timestamp"`
	DeviceID  string `db:"deviceid" json:"-"`
	IPAddr    string `db:"ipAddr" json:"-"`
	PostMeta
}

type PostMeta struct {
	CommentsCount int        `db:"comments" json:"comments_count"`
	LikesCount    int        `db:"likes" json:"likes_count"`
	Comments      []*Comment `json:"comments"`
}

type Comment struct {
	ID        int    `db:"commentid" json:"commentid"`
	Text      string `db:"comment" json:"comment"`
	Timestamp int64  `db:"timestamp" json:"timestamp"`
	DeviceID  string `db:"deviceid" json:"-"`
}

type Like struct {
	DeviceID string `db:"deviceid" json:"-"`
}
