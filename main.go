package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/envelope-app/envelope-backend/db"
	"github.com/envelope-app/envelope-backend/log"
	"github.com/envelope-app/envelope-backend/router"
)

var port = os.Getenv("PORT")

func main() {
	log.Info.Printf("Starting Envelope Backend...\n")

	if port == "" {
		log.Error.Fatalln("$PORT not set")
	}

	dbs, err := db.Init()
	if err != nil {
		log.Error.Fatalf("%v: %s", err, err)
	}

	router := router.Init(dbs)

	err = http.ListenAndServe(fmt.Sprintf(":%s", port), router)

	if err != nil {
		log.Error.Fatalln("%v: %s", err, err)
	}
}
