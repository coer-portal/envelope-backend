# Envelope Backend

This project is the backend of an Anonymous Facebook like platform. The application is called [Envelope](https://github.com/envelope-app/envelope-android).

The Backend is written in [Go](https://golang.org). It uses Postgresql to store posts/likes/comments with a redis proxy that caches recent posts.

The compiled executable is packed in a Docker container and deployed to EC2.

# Requirements
1. Golang Compiler. Download a compatible version for your machine from [here](golang.org/dl)
2. Postgresql
3. Redis

# Build Instructions

Before, Building, Be sure to set up environment variables correctly. If you are using Docker, Put correct values in setup_env file.

    git clone https://github.com/envelope-app/envelope-backend
    cd envelope-backend
    go get github.com/envelope-app/envelope-backend
    go build

Compiling proto service descriptor file

    protoc --go_out=plugins=grpc:envelope/ *.proto

We prefer a multi stage docker container for docker based deployments.

    docker build -t envelope .
    docker run --rm -it --env-file setup_env --net=host envelope

# Architecture
	// TODO


## Contributors

1. Ishan Jain([@ishanjain28](https://github.com/ishanjain28))
2. Mrinal Raj([@mrinalraj](http://github.com/mrinalraj))
3. Piyush Bhatt([@piyush01bhatt](https://github.com/Piyush01Bhatt))

# License
MIT

