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


We prefer a multi stage docker container for docker based deployments. 

    docker build -t envelope . 
    docker run --rm -it --env-file setup_env --net=host envelope

# Architecture
	// TODO

# API Documentation

All endpoints, return a JSON response. `Accept` header is not obeyed. 

Unless specified seperately, 
1. `deviceid` refers to the device id of request origin, i.e. a unique id of every device. 

## Submit Post
#### Request 

Endpoint

    POST /submit-post

Headers

    Content-Type: application/x-www-form-urlencoded

Body

    text: <The text content of post>


#### Response 
##### Successful 

    {
        "share_hash":"55de90e87fcc1257",
        "edit_hash":"b4ff21354b051367efba4ed48afe73181fccfc93b1746f55e9f658e260e1891a",
        "time":1503590894
    }

##### Fail

    {
	    error: err.message,
		code: code
	}

### Report Post

#### Request 

Endpoint

    POST /report-post

Headers

    Content-Type: application/x-www-form-urlencoded
    deviceid: ""

#### Response 
	//TODO
	
###  Fetch Post Meta
#### Request 

Endpoint

    GET /fetch-post-meta

Headers

    Content-Type: application/x-www-form-urlencoded

#### Response 
	//TODO

###  Edit Post 
#### Request 

Endpoint

    POST /edit-post

Headers

    Content-Type: application/x-www-form-urlencoded
    deviceid: ""
	edit_hash: <edit hash of post, This is required for a successfull edit>

#### Response
	//TODO
	
### Like Post
#### Request 

Endpoint

    POST /like-post

Headers

    Content-Type: application/x-www-form-urlencoded
   
#### Response
	//TODO

### Submit Comment
#### Request 

Endpoint

    POST /submit-comment

Headers

    Content-Type: application/x-www-form-urlencoded

#### Response
	//TODO

	
### Fetch Posts
#### Request 

Endpoint

    GET /fetch-posts

Headers

#### Response
	//TODO


## Contributors
	
1. Ishan Jain([@ishanjain28](https://github.com/ishanjain28))
2. Mrinal Raj([@mrinalraj](http://github.com/mrinalraj))
3. Piyush Bhatt([@piyush01bhatt](https://github.com/Piyush01Bhatt))

# License
MIT

