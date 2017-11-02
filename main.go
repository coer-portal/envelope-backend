package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-redis/redis"
	"github.com/ishanjain28/envelope-backend/db"
	"github.com/ishanjain28/envelope-backend/log"
	"github.com/ishanjain28/envelope-backend/router"
)

var port = os.Getenv("PORT")
var redisAddr = os.Getenv("REDIS_SERVER")

func main() {
	log.Info.Printf("Starting Envelope Backend...\n")

	if port == "" {
		log.Error.Fatalln("$PORT not set")
	}

	if redisAddr == "" {
		log.Error.Fatalln("$REDIS_SERVER not set")
	}

	redisOpt, err := redis.ParseURL(redisAddr)
	if err != nil {
		log.Error.Fatalf("Invalid $REDIS_SERVER: %v\n", err)
	}

	client := redis.NewClient(redisOpt)

	err = client.Ping().Err()
	if err != nil {
		log.Error.Fatalf("Error in connecting to redis: %s", err)
	}

	pqdb, err := db.Init()
	if err != nil {
		log.Error.Fatalf("%v: %s", err, err)
	}

	router := router.Init(client, pqdb)

	err = http.ListenAndServe(fmt.Sprintf(":%s", port), router)

	if err != nil {
		log.Error.Fatalln("%v: %s", err, err)
	}
}
