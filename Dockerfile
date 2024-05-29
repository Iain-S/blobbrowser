FROM golang:1.22.3-alpine3.20

WORKDIR /code

COPY go.mod go.sum main.go handlers.go ./

RUN go build

RUN rm go.mod go.sum main.go handlers.go

ENTRYPOINT ["./browser"]
