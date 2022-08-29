FROM golang:1.18 AS builder

COPY . /go/src/app
WORKDIR /go/src/app

ENV GO111MODULE=on

RUN go install github.com/swaggo/swag/cmd/swag@v1.8.4
RUN swag init --parseDependency -d ./pkg/api -g api.go
RUN CGO_ENABLED=0 GOOS=linux go build -o app

RUN git log -1 --oneline > version.txt

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /go/src/app/app .
COPY --from=builder /go/src/app/config.json .
COPY --from=builder /go/src/app/version.txt .

EXPOSE 8080

ENTRYPOINT ["./app"]