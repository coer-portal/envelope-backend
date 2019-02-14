package main

import (
	"net"
	"os"

	"github.com/envelope-app/envelope-backend/db"
	"github.com/envelope-app/envelope-backend/envelope"
	"github.com/envelope-app/envelope-backend/log"
	"github.com/envelope-app/envelope-backend/router"
	"google.golang.org/grpc"
)

var port = os.Getenv("PORT")

func main() {
	log.Info.Println("Starting Envelope Backend...")
	if port == "" {
		log.Error.Fatalln("$PORT not set")
	}

	dbs, err := db.Init()
	if err != nil {
		log.Error.Fatalf("%v: %s", err, err)
	}

	router := router.Init(dbs)

	conn, err := net.Listen("tcp", port)
	if err != nil {
		log.Error.Fatalln("error in listening on port", port)
	}

	grpcConn := grpc.NewServer()

	envelope.RegisterEnvelopeServer(grpcConn, router)

	if err := grpcConn.Serve(conn); err != nil {
		log.Error.Fatalf("failed to serve: %v\n", err)
	}
}
