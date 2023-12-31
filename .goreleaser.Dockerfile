FROM alpine:3.18 as builder

RUN apk --no-cache add ca-certificates

WORKDIR /app/
COPY . .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]
