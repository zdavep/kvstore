FROM golang:1.13-buster as build

WORKDIR /go/src/github.com/zdavep/kvstore

RUN apt-get update && apt-get upgrade -y && apt-get install -y libleveldb-dev

COPY go.* ./
COPY cmd ./cmd/
COPY internal ./internal/
RUN go build -tags cleveldb -o /go/bin/kvstore cmd/kvstore/main.go

FROM debian:buster-slim
RUN apt-get update && apt-get upgrade -y && apt-get install -y libleveldb-dev
COPY --from=build /go/bin/kvstore /
ENTRYPOINT ["/kvstore"]
