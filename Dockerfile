FROM golang:1.22.3-alpine3.20

WORKDIR /code

COPY home.html login.html ./
COPY go.mod go.sum handlers.go main.go settings.go utils.go  ./

RUN go build

RUN rm go.mod go.sum handlers.go main.go settings.go utils.go

ENTRYPOINT ["./browser"]
