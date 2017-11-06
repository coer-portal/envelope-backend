package router

var OK = "OK"

type OkResponse struct {
	Status string `json:"status"`
}

type RegisterDeviceResponse struct {
	Status string `json:"status"`
	Hash   string `json:"hash"`
}

type SubmitPostResponse struct {
	Status    string `json:"status"`
	PostID    int64  `json:"postid"`
	Timestamp int64  `json:"timestamp"`
	likes     int    `json:"likes"`
}
