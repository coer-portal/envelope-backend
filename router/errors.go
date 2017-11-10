package router

var (

	// ErrInvalidData is sent when a value in request is invalid
	ErrInvalidData = "INVALID_DATA"
	// ErrInternal is send when a internal server error occurs.
	ErrInternal = "INTERNAL_ERROR"
	// ErrParsing is sent when an error occurs in parsing the request
	ErrParsing = "PARSING_ERROR"
	// ErrOutOfValidRegion occurs when the user is out of region we previously decided
	ErrOutOfValidRegion = "OUT_OF_REGION"
	// ErrNotRegistered is sent when a deviceid is not registered
	ErrNotRegistered = "NOT_REGISTERED"
	// ErrNotFound is sent when a expected value is missing from request
	ErrNotFound = "NOT_FOUND"

	// ErrTimeout is sent when a request's context deadline is exceeded or if it is canceled
	ErrTimeout = "TIMEOUT"

	ErrExpired = "EXPIRED"
)
