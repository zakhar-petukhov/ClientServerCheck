# Golang Image
FROM golang:1.12.7-alpine3.10 AS build

RUN apk --no-cache add gcc g++ make
RUN apk add git

WORKDIR /go/src/app
COPY . .

RUN go get github.com/gorilla/websocket
RUN GOOS=linux go build -ldflags="-s -w" -o ./bin/test ./main.go


# Alpine Image
FROM alpine:3.10

RUN apk --no-cache add ca-certificates

WORKDIR /usr/bin
COPY . .
COPY --from=build /go/src/app/bin /go/bin

EXPOSE 4567
ENTRYPOINT /go/bin/test --port 4567