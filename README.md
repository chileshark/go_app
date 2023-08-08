# go_app
FROM golang:1.20.7-alpine
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /home/golang/pings
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY main.go .
RUN go build -o latency_ping .

FROM alpine:latest
COPY --from=golang_ping /home/golang/pings/latency_ping /
COPY --from=golang_ping /home/golang/pings/config.ini /
CMD ["./latency_ping","-a","config.ini"]

###RUN
docker run -it --rm --name ping_server -v /root/golangcode/newping/config.ini:/config.ini ping_app
