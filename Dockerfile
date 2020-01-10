FROM golang:1.13 as build

WORKDIR /go/src/github.com/zdavep/kvstore

COPY go.* ./
COPY cmd ./cmd/
COPY internal ./internal/
RUN go build -o /go/bin/kvstore cmd/kvstore/main.go

FROM gcr.io/distroless/base:debug
COPY --from=build /go/bin/kvstore /
ENTRYPOINT ["/kvstore"]
