
FROM golang:alpine

RUN apk update
RUN apk add ca-certificates git 

ARG workdir=/go/src/github.com/ishanjain28/envelope-backend
 
COPY . $workdir 
WORKDIR $workdir

RUN go get github.com/envelope-app/envelope-backend

RUN go install 

FROM alpine

RUN apk update
RUN apk add ca-certificates

COPY --from=0 /go/bin/envelope-backend /usr/bin/envelope-backend

CMD /usr/bin/envelope-backend
