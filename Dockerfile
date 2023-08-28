FROM golang:1.21.0-alpine as builder

RUN apk --no-cache add ca-certificates

WORKDIR /app/
COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go mod download
RUN go test -v ./...
RUN go build -o app

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /user/app /app

ENTRYPOINT ["/app"]
