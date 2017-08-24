# envelope-backend

Envelope is a Anonymous Text/Photo Posting website for College Students. 
This repository has the code for the server 

## Endpoints

### Submit Text Post

#### Request 

Endpoint

    POST /submit-post

Headers

    Content-Type: application/x-www-form-urlencoded
    Accept: <The Type of response that you want>, e.g. */*, application/json, text/html

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

    When `Accept` header is set to `*/*` or `application/json`

    {
	    error: err.message,
		code: code
	}

When `Accept` header is set to anything other than `*/*` or `applcation/json`

    <html>
	<head>
		<title>Error Occurred</title>
	</head>
	<body>
		<h1>Error Occurred: ${code}</h1>
		<p>${message}<p>
	</body>
    </html>


## Contributors
	
1. Ishan Jain([@ishanjain28](https://github.com/ishanjain28))
2. Mrinal Raj([@mrinalraj](http://github.com/mrinalraj))
